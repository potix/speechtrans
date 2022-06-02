
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
	writeMutex sync.Mutex
}

type HttpHandler struct {
        verbose      bool
        resourcePath string
        accounts     map[string]string
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

func (h *HttpHandler) clientRegister(conn *websocket.Conn) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	h.clients[conn] = new(client)
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

func (h *HttpHandler) translationLoop(conn *websocket.Conn) {
	h.clientRegister(conn)
	defer h.clientUnregister(conn)
	defer conn.Close()
	pingStopCh := make(chan int)
	go h.startPingLoop(conn, pingStopCh)
	defer close(pingStopCh)

	// XXX new translator

	for {

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

func NewHttpHandler(resourcePath string, accounts map[string]string, opts ...HttpOption) (*HttpHandler, error) {
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
        }, nil
}
