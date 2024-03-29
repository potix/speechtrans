
package handler

import (
        "log"
        "fmt"
        "path"
        "net/http"
	"sync"
	"encoding/json"
	"time"
        "github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/potix/speechtrans/message"
	"github.com/potix/speechtrans/translator"
)

type httpOptions struct {
        verbose bool
}

func defaultHttpOptions() *httpOptions {
        return &httpOptions {
                verbose: false,
        }
}

type HttpOption func(*httpOptions)

func HttpVerbose(verbose bool) HttpOption {
        return func(opts *httpOptions) {
                opts.verbose = verbose
        }
}

type client struct {
	writeMutex      sync.Mutex
	progressInAudio bool
	translator      *translator.Translator
}

type HttpHandler struct {
        verbose      bool
        resourcePath string
        accounts     map[string]string
	projectId    string
	clientsMutex sync.Mutex
	clients      map[*websocket.Conn]*client
}

func (h *HttpHandler) Start() error {
	return nil
}

func (h *HttpHandler) Stop() {
}

func (h *HttpHandler) SetRouting(router *gin.Engine) {
	favicon := path.Join(h.resourcePath, "icon", "favicon.ico")
        js := path.Join(h.resourcePath, "js")
        css := path.Join(h.resourcePath, "css")
        img := path.Join(h.resourcePath, "img")
        font := path.Join(h.resourcePath, "font")
	templatePath := path.Join(h.resourcePath, "template", "*")
        router.LoadHTMLGlob(templatePath)
	authGroup := router.Group("/", gin.BasicAuth(h.accounts))
	authGroup.GET("/", h.indexHtml)
	authGroup.GET("/index.html", h.indexHtml)
	authGroup.GET("/ws/trans", h.translation)
	authGroup.StaticFile("/favicon.ico", favicon)
        authGroup.Static("/js", js)
        authGroup.Static("/css", css)
        authGroup.Static("/img", img)
        authGroup.Static("/font", font)
}

func (h *HttpHandler) indexHtml(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func (h *HttpHandler) clientRegister(conn *websocket.Conn) *client {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	tVerboseOpt := translator.TranslatorVerbose(h.verbose)
	newTranslator := translator.NewTranslator(h.projectId, tVerboseOpt)
	h.clients[conn] = &client {
		progressInAudio: false,
		translator: newTranslator,
	}
	return h.clients[conn]
}

func (h *HttpHandler) clientUnregister(conn *websocket.Conn) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	delete(h.clients, conn)
}

func (h *HttpHandler) getClient(conn *websocket.Conn) (*client, error) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	client, ok := h.clients[conn]
	if !ok {
		return nil, fmt.Errorf("not found client %v", conn)
	}
	return client, nil
}

func (h *HttpHandler) safeWriteMessage(conn *websocket.Conn, messageType int, message []byte) error {
	client, err := h.getClient(conn)
	if err != nil {
		return fmt.Errorf("can not get client, write failure %v, %v, %v: %w", conn, messageType, message, err)
	}
	client.writeMutex.Lock()
	defer client.writeMutex.Unlock()
	return conn.WriteMessage(messageType, message)
}

func (h *HttpHandler) startPingLoop(conn *websocket.Conn, pingLoopStopChan chan int) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			msg := &message.Message{
				MType: message.MTypePing,
			}
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				log.Printf("can not marshal to json: %v", err)
				break
			}
			err = h.safeWriteMessage(conn, websocket.TextMessage, msgBytes)
			if err != nil {
				log.Printf("can not write ping message: %v", err)
				return
			}
		case <-pingLoopStopChan:
			return
		}
	}
}

func (h *HttpHandler) sendEmptyMessage(conn *websocket.Conn, mType string, errorMessage string) error {
	newMsg := &message.Message{
		MType: mType,
	}
	if errorMessage != "" {
		newMsg.Error = &message.Error{ Message: errorMessage }
	}
	newMsgJson, err := json.Marshal(newMsg)
	if err != nil {
		return fmt.Errorf("can not marshal empty message to json: %w", err)
	}
	err = h.safeWriteMessage(conn, websocket.TextMessage, newMsgJson)
	if err != nil {
		return fmt.Errorf("can not write empty message: %w", err)
	}
	return nil
}

func (h *HttpHandler) toTextNotifyCb(conn *websocket.Conn, err error) {
	if err == nil {
		return
	}
	newMsg := &message.Message {
		MType: message.MTypeToTextNotify,
		Error: &message.Error{
			Message: fmt.Sprintf("%v", err),
		},
	}
	newMsgJson, err := json.Marshal(newMsg)
	if err != nil {
		log.Printf("can not marshal toText notify message to json: %v", err)
	}
	err = h.safeWriteMessage(conn, websocket.TextMessage, newMsgJson)
	if err != nil {
		log.Printf("can not write toText notify message: %v", err)
	}
}

func (h *HttpHandler) translationLoop(conn *websocket.Conn) {
	client := h.clientRegister(conn)
	defer h.clientUnregister(conn)
	defer conn.Close()
	pingStopCh := make(chan int)
	go h.startPingLoop(conn, pingStopCh)
	defer close(pingStopCh)
	defer client.translator.Cleanup()
	for {
		t, msgJson, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if t != websocket.TextMessage {
			log.Printf("unsupported message type: %v", t)
			continue
		}
		var msg message.Message
		if err := json.Unmarshal(msgJson, &msg); err != nil {
			log.Printf("can not unmarshal message: %v", err)
			continue
		}
		if msg.MType == message.MTypePing {
			continue
		} else if msg.MType == message.MTypeInAudioConfReq {
			if msg.InAudioConf == nil            ||
			   msg.InAudioConf.Encoding == ""    ||
			   msg.InAudioConf.SampleRate == 0   ||
			   msg.InAudioConf.SampleSize == 0   ||
			   msg.InAudioConf.ChannelCount == 0 ||
			   msg.InAudioConf.SrcLang == ""     {
				err := h.sendEmptyMessage(conn, message.MTypeInAudioConfRes, "invalid argument")
				if err != nil {
					log.Printf("can not write inAudioConfRes message: %v", err)
					continue
				}
			}
			if client.progressInAudio {
				err := h.sendEmptyMessage(conn, message.MTypeInAudioConfRes, "already tranlate in progess")
				if err != nil {
					log.Printf("can not write inAudioConfRes message: %v", err)
					continue
				}
			}
			log.Printf("inAudioConfReq %+v", msg.InAudioConf)
			client.translator.ToText(conn, msg.InAudioConf, h.toTextNotifyCb)
			client.progressInAudio = true
			err := h.sendEmptyMessage(conn, message.MTypeInAudioConfRes, "")
			if err != nil {
				log.Printf("can not write inAudioConfRes message: %v", err)
				continue
			}
		} else if msg.MType == message.MTypeInAudioDataReq {
			if msg.InAudioData == nil ||
			   len(msg.InAudioData.DataBytes) == 0 {
				err := h.sendEmptyMessage(conn, message.MTypeInAudioDataRes, "invalid argument")
				if err != nil {
					log.Printf("can not write inAudioDataRes message: %v", err)
					continue
				}
			}
			if !client.progressInAudio {
				continue
			}
			client.translator.ToTextContent(msg.InAudioData.DataBytes)
			err := h.sendEmptyMessage(conn, message.MTypeInAudioDataRes, "")
			if err != nil {
				log.Printf("can not write inAudioDataRes message: %v", err)
				continue
			}
		} else if msg.MType == message.MTypeInAudioDataEndReq {
			if !client.progressInAudio {
				continue
			}
			log.Printf("inAudioDataEndReq")
			client.translator.ToTextContentEnd()
			client.progressInAudio = false
			err := h.sendEmptyMessage(conn, message.MTypeInAudioDataEndRes, "")
			if err != nil {
				log.Printf("can not write inAudioDataEndRes message: %v", err)
				continue
			}
		} else if msg.MType == message.MTypeTranslateReq {
			if msg.TransConf == nil        ||
			   msg.TransConf.SrcLang == "" ||
			   msg.TransConf.DstLang == "" ||
			   msg.TransConf.Gender == "" {
				err := h.sendEmptyMessage(conn, message.MTypeTranslateRes, "invalid argument")
				if err != nil {
					log.Printf("can not write translateRes message: %v", err)
					continue
				}
			}
			log.Printf("translateReq %+v", msg.TransConf)
			outAudioDataBytes, outAudioEncoding, err := client.translator.Translate(msg.TransConf)
			if err != nil {
				err := h.sendEmptyMessage(conn, message.MTypeTranslateRes, fmt.Sprintf("can not translate: %v", err))
				if err != nil {
					log.Printf("can not write translateRes message: %v", err)
					continue
				}
			}
			newMsg := &message.Message {
				MType: message.MTypeTranslateRes,
				TransResult: &message.TransResult{
					Encoding: outAudioEncoding,
					DataBytes: outAudioDataBytes,
				},
			}
			newMsgJson, err := json.Marshal(newMsg)
			if err != nil {
				log.Printf("can not marshal translate response message to json: %v", err)
				continue
			}
			err = h.safeWriteMessage(conn, websocket.TextMessage, newMsgJson)
			if err != nil {
				log.Printf("can not write translate response message: %v", err)
				continue
			}
		} else {
			log.Printf("unsupported message type: %v", msg.MType)
		}
	}
}

func (h *HttpHandler) translation(c *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		Subprotocols: []string{"translation"},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade: %+v", err)
                c.AbortWithStatus(400)
		return
	}
	go h.translationLoop(conn)
}

func NewHttpHandler(resourcePath string, accounts map[string]string, projectId string, opts ...HttpOption) (*HttpHandler, error) {
        baseOpts := defaultHttpOptions()
        for _, opt := range opts {
                if opt == nil {
                        continue
                }
                opt(baseOpts)
        }
	return &HttpHandler{
                verbose:          baseOpts.verbose,
                resourcePath:     resourcePath,
                accounts:         accounts,
		projectId:        projectId,
		clients:          make(map[*websocket.Conn]*client),
        }, nil
}


