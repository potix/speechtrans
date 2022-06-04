package handler

import (
	"log"
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
}


func (t *Translator) SpeechToTextStart(inAudioConfig *message.InAudioConfig, resultCallback func([]bytes)) {
	go func() {
	        t.speechToText := new speechToText()
		srcText, err := t.speechToText.StreamingRecognizeRequestStart(inAudioConfig) //blocking
		dstText := t.translateText(srcText)
		textToSpeech = new TextToSpeech()
		textToSpeech
	}
}

func (t *Translator) SpeechToTextContent([]bytes) {
	t.speechToText.StreamingRecognizeRequestContent([]byte)
}

func (t *Translator) SpeechToTexContentEnd() {
	t.speechToText.StreamingRecognizeRequestContentEnd()
}


func NewTranslator(opts ...TranslatorOption) *Translator {
	baseOpts := defaultTranslatorOptions()
        for _, opt := range opts {
                if opt == nil {
                        continue
                }
                opt(baseOpts)
        }
	return &Translator{
		opts:            baseOpts,
		toTcpChan:       make(chan *GamepadMessage),
		toWsChan:        make(chan *GamepadMessage),
		stopFromTcpChan: make(chan int),
		stopFromWsChan:  make(chan int),
		started:         false,
		speechToTextCtx  null,
	}
}


type SpeachToTextStream struct {
i	client
	stream
}

func (s SpeachToTextStream) StreamingRecognizeRequestStart(resultcallback func()) SpeachToTextStream {
	ctx := context.Background()

        client, err := speech.NewClient(ctx)
        if err != nil {
                log.Fatal(err)
        }
        stream, err := client.StreamingRecognize(ctx)
        if err != nil {
                log.Fatal(err)
        }

        // Send the initial configuration message.
        if err := stream.Send(&speechpb.StreamingRecognizeRequest{
                StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
                        StreamingConfig: &speechpb.StreamingRecognitionConfig{
                                Config: &speechpb.RecognitionConfig{
                                        Encoding:        speechpb.RecognitionConfig_LINEAR16,
                                        SampleRateHertz: 16000,
                                        LanguageCode:    "en-US",
                                },
                        },
                },
        }); err != nil {
                log.Fatal(err)
        }

	contentRequestch = make(chan)

	go func() {
		for {
                        buff, ok <- contentRequestCh
			if !ok {

                                // Nothing else to pipe, close the stream.
                                if err := stream.CloseSend(); err != nil {
                                        log.Fatalf("Could not close stream: %v", err)
                                }
				return
			}

                        if err := stream.Send(&speechpb.StreamingRecognizeRequest{
			       StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
					AudioContent: buf[:n],
                                        },
			}); err != nil {
				log.Printf("Could not send audio: %v", err)
			}

                }
        }()

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Cannot stream results: %v", err)
			}
			if err := resp.Error; err != nil {
                        // Workaround while the API doesn't give a more informative error.
                        if err.Code == 3 || err.Code == 11 {
                                log.Print("WARNING: Speech recognition request exceeded limit of 60 seconds.")
                        }
                        log.Fatalf("Could not recognize: %v", err)
			}
			for _, result := range resp.Results {
				fmt.Printf("Result: %+v\n", result)

				resultCallback()
			}
		}

}

func (s SpeachToTextStream) StreamingRecognizeContentRequest([]byte) (timeLimit bool) {
	 conten Requestch <- bytes	
}

func (s SpeachToTextStream) StreamingRecognizStop([]byte){
	close(contentRequestch) 	
}


:w

func transfer() {



	func translateText(projectID string, sourceLang string, targetLang string, contents []string) error {
        ctx := context.Background()
        client, err := translate.NewTranslationClient(ctx)
        if err != nil {
            return fmt.Errorf("NewTranslationClient: %v", err)
        }
        defer client.Close()

        req := &translatepb.TranslateTextRequest{
            Parent: fmt.Sprintf("projects/%s/locations/global", projectID),
            SourceLanguageCode: sourceLang,
            TargetLanguageCode: targetLang,
            MimeType:           "text/plain", // Mime types: "text/plain", "text/html"
            Contents:           contents,
        }

        resp, err := client.TranslateText(ctx, req)
            if err != nil {
                return fmt.Errorf("TranslateText: %v", err)
        }

        // Display the translation for each input text provided
        for _, translation := range resp.GetTranslations() {
            fmt.Printf("Translated text: %v\n", translation.GetTranslatedText())
        }

        return nil
}

func main() {
         translateText("clipper-255807", "en-us", "ja", []string{"hello, world!", "I am hungry."})
}
}
