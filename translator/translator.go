package translator

import (
	"log"
	"fmt"
	"strings"
	"context"
	"io"

	"github.com/potix/speechtrans/message"
	"github.com/gorilla/websocket"

	"google.golang.org/protobuf/types/known/wrapperspb"
	speech "cloud.google.com/go/speech/apiv1"
        speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	translate "cloud.google.com/go/translate/apiv3"
	translatepb "google.golang.org/genproto/googleapis/cloud/translate/v3"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
        texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

type translatorOptions struct {
	verbose bool
}

func defaultTranslatorOptions() *translatorOptions {
	return &translatorOptions {
		verbose: false,
	}
}

type TranslatorOption func(*translatorOptions)

func TranslatorVerbose(verbose bool) TranslatorOption {
        return func(opts *translatorOptions) {
                opts.verbose = verbose
        }
}

type Translator struct {
	opts            *translatorOptions
	projectId        string
	progressInAudio  bool
	inAudioDataCh    chan []byte
	toTextCompWaitCh chan int
	srcText          []string
	dstText          []string
	voiceMap         map[string]string
}

func (t *Translator)speechToText(ctx context.Context, inAudioConf *message.InAudioConf, inAudioDataCh chan []byte) (error) {
        client, err := speech.NewClient(ctx)
        if err != nil {
		return fmt.Errorf("can not create speech client: %v", err)
        }
	defer client.Close()
        stream, err := client.StreamingRecognize(ctx)
        if err != nil {
		return fmt.Errorf("can not create stream: %v", err)
        }
	log.Printf(">>> %v, %v, %v", inAudioConf.SampleRate, inAudioConf.SrcLang, inAudioConf.ChannelCount)
        // Send the initial configuration message.
        err = stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:          speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz:   inAudioConf.SampleRate,
					AudioChannelCount: inAudioConf.ChannelCount,
					LanguageCode:      inAudioConf.SrcLang,
					MaxAlternatives:   1,
					EnableAutomaticPunctuation: true,
					EnableSpokenPunctuation: wrapperspb.Bool(true),
					EnableSpokenEmojis: wrapperspb.Bool(false),
					Model: "default",
					UseEnhanced: false,
				},
			},
		},
        })
	if err != nil {
		return fmt.Errorf("can not send config of streaming recognize request: %v", err)
        }
	go func() {
		for {
			buff, ok := <-inAudioDataCh
			if !ok {
                                // Nothing else to pipe, close the stream.
                                if err := stream.CloseSend(); err != nil {
                                        log.Printf("can not close stream: %v", err)
                                }
				return
			}
			//log.Printf("recv input audio data (len = %v)", len(buff))
			err := stream.Send(&speechpb.StreamingRecognizeRequest{
				StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
					AudioContent: buff,
				},
			})
			if err != nil {
				log.Printf("can not send content of streaming recognize request: %v", err)
			}
                }
        }()
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("can not get results from stream: %v", err)
		}
		if err := resp.Error; err != nil {
			// Workaround while the API doesn't give a more informative error.
			if err.Code == 3 || err.Code == 11 {
				log.Print("WARNING: Speech recognition request exceeded limit of 60 seconds.")
			}
			return fmt.Errorf("can not recognize: %v", err)
		}
		for _, result := range resp.Results {
			for _, alternative := range result.Alternatives {
				t.srcText = append(t.srcText, alternative.Transcript)
                        }
		}
	}
	return nil
}

func (t *Translator) ToText(conn *websocket.Conn, inAudioConf *message.InAudioConf, toTextNotifyCb func(*websocket.Conn, error)) {
	if t.progressInAudio {
		return
	}
	t.progressInAudio = true
	t.inAudioDataCh = make(chan []byte)
	t.toTextCompWaitCh = make(chan int)
	go func() {
		ctx := context.Background()
		err := t.speechToText(ctx, inAudioConf, t.inAudioDataCh)
		if err != nil {
			toTextNotifyCb(conn, fmt.Errorf("can not convert audio to text: %w", err))
		}
		close(t.toTextCompWaitCh)
	}()
}

func (t *Translator) ToTextContent(dataBytes []byte) {
	if !t.progressInAudio {
		return
	}
	t.inAudioDataCh <- dataBytes
	//log.Printf("send input audio data (len = %v)", len(t.inAudioDataCh))
}

func (t *Translator) ToTextContentEnd() {
	if !t.progressInAudio {
		return
	}
	close(t.inAudioDataCh)
	<-t.toTextCompWaitCh
	t.inAudioDataCh = nil
	t.progressInAudio = false
}

func (t *Translator) translateText(ctx context.Context, transConf *message.TransConf) error {
        client, err := translate.NewTranslationClient(ctx)
        if err != nil {
            return fmt.Errorf("can not create translation client: %v", err)
        }
        defer client.Close()
        req := &translatepb.TranslateTextRequest{
            Parent: fmt.Sprintf("projects/%s/locations/global", t.projectId),
            SourceLanguageCode: transConf.SrcLang,
            TargetLanguageCode: transConf.DstLang,
            MimeType:           "text/plain",
            Contents:           t.srcText,
        }
        resp, err := client.TranslateText(ctx, req)
            if err != nil {
                return fmt.Errorf(" can not translate text: %v", err)
        }
        for _, translation := range resp.GetTranslations() {
		t.dstText = append(t.dstText, translation.GetTranslatedText())
        }
	return nil
}

func  (t *Translator) textToSpeech(ctx context.Context, transConf *message.TransConf) ([]byte, string, error) {
        client, err := texttospeech.NewClient(ctx)
        if err != nil {
		return []byte{}, "", fmt.Errorf("can not create text to speech client: %v", err)
        }
        defer client.Close()
	name := ""
	voiceName, ok := t.voiceMap[strings.ToLower(transConf.DstLang + ":" + transConf.Gender)]
	if ok {
		name = voiceName
	}
	ssmlGender := texttospeechpb.SsmlVoiceGender_MALE
	if transConf.Gender == "female" {
		ssmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
	}
        req := texttospeechpb.SynthesizeSpeechRequest{
                Input: &texttospeechpb.SynthesisInput{
                        InputSource: &texttospeechpb.SynthesisInput_Text{Text: strings.Join(t.dstText, "\n")},
                },
                Voice: &texttospeechpb.VoiceSelectionParams{
                        LanguageCode: transConf.DstLang,
			Name: name,
			SsmlGender: ssmlGender,
                },
                AudioConfig: &texttospeechpb.AudioConfig{
                        AudioEncoding: texttospeechpb.AudioEncoding_OGG_OPUS,
                },
        }
        resp, err := client.SynthesizeSpeech(ctx, &req)
        if err != nil {
		return []byte{}, "", fmt.Errorf("can not synthesize speech: %v", err)
        }
	return resp.AudioContent, message.EncodingOggOpus, nil
}

func (t *Translator) Translate(transConf *message.TransConf) ([]byte, string, error) {
	defer func() {
		t.srcText = t.srcText[:0]
		t.dstText = t.dstText[:0]
	}()
	ctx := context.Background()
	err := t.translateText(ctx, transConf)
	if err != nil {
		return  []byte{}, "", fmt.Errorf("can not translate: %w", err)
	}
	outAudioDataBytes, outAudioEncoding, err := t.textToSpeech(ctx, transConf)
	if err != nil {
		return  []byte{}, "", fmt.Errorf("can not convert text to audio: %v", err)
	}
	return outAudioDataBytes, outAudioEncoding, nil
}

func (t *Translator) Cleanup() {
	if t.inAudioDataCh != nil {
		close(t.inAudioDataCh)
	}
	t.srcText = t.srcText[:0]
	t.dstText = t.dstText[:0]
}

func NewTranslator(projectId string, opts ...TranslatorOption) *Translator {
	baseOpts := defaultTranslatorOptions()
        for _, opt := range opts {
                if opt == nil {
                        continue
                }
                opt(baseOpts)
        }
	voiceMap := map[string]string {
		"ja-jp:male":   "ja-JP-Standard-D",
		"ja-jp:female": "ja-JP-Standard-A",
		"en-us:male":   "en-US-Standard-B",
		"en-us:female": "en-US-Standard-G",
		"en-gb:male":   "en-GB-Standard-B",
		"en-gb:female": "en-GB-Standard-A",
		"en-au:male":   "en-AU-Standard-B",
		"en-au:female": "en-AU-Standard-A",
		"fr-fr:male":   "fr-FR-Standard-B",
		"fr-fr:female": "fr-FR-Standard-A",
		"nl-nl:male":   "nl-NL-Standard-B",
		"nl-nl:female": "nl-NL-Standard-A",
		"de-de:male":   "de-DE-Standard-B",
		"de-de:female": "de-DE-Standard-A",
		"it-it:male":   "it-IT-Standard-C",
		"it-it:female": "it-IT-Standard-A",
		"ko-KR:male":   "ko-KR-Standard-C",
		"ko-KR:female": "ko-KR-Standard-A",
		"ru-RU:male":   "ru-RU-Standard-B",
		"ru-RU:female": "ru-RU-Standard-A",
		"sv-SE:male":   "sv-SE-Standard-B",
		"sv-SE:female": "sv-SE-Standard-A",
		"tr-TR:male":   "tr-TR-Standard-B",
		"tr-TR:female": "tr-TR-Standard-A",
	}
	return &Translator{
		opts:            baseOpts,
		projectId:       projectId,
		progressInAudio: false,
		inAudioDataCh:   nil,
		srcText:         make([]string, 0, 10),
		dstText:         make([]string, 0, 10),
		voiceMap:        voiceMap,
	}
}


