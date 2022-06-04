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
	progressInAudio bool
	inAudioDataCh   chan []byte
	srcText         []string
	dstText         []string
}

func (t *Translator) speechToText(ctx context.Context, inAudioConf *message.InAudioConf, inAudioDataCh chan []byte) (srcText string, error) {
        client, err := speech.NewClient(ctx)
        if err != nil {
		return "", fmt.Errorf("can not create speech client: %v", err)
        }
	defer client.Close()
        stream, err := client.StreamingRecognize(ctx)
        if err != nil {
		return "", fmt.Errorf("can not create stream: %v", err)
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
					EnableSpokenPunctuation: true,
					EnableSpokenEmojis: false,
					Model: "default",
					UseEnhanced: false,

				},
			},
		},
        })
	if err != nil {
		return "", fmt.Errorf("can not send config of streaming recognize request: %v", err)
        }
	go func() {
		for {
                        buff, ok <-inAudioDataCh
			if !ok {
                                // Nothing else to pipe, close the stream.
                                if err := stream.CloseSend(); err != nil {
                                        log.Printf("can not close stream: %v", err)
                                }
				return
			}
			err := stream.Send(&speechpb.StreamingRecognizeRequest{
				StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
					AudioContent: buf,
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
			return "", fmt.Errorf("can not get results from stream: %v", err)
		}
		if err := resp.Error; err != nil {
			// Workaround while the API doesn't give a more informative error.
			if err.Code == 3 || err.Code == 11 {
				log.Print("WARNING: Speech recognition request exceeded limit of 60 seconds.")
			}
			return "", fmt.Errorf("can not recognize: %v", err)
		}
		for _, result := range resp.Results {
			for _, alternative := range result.Alternatives {
				t.srcText = append(t.srcText, alternative.Transcript)
                        }
		}
	}
}

func (t *Translator) ToText(conn *websocket.Conn, inAudioConf *message.InAudioConf, toTextNotifyCb func(*websocket.Conn, error)) {
	if t.progressInAudio {
		return
	}
	t.progressInAudio = true
	t.inAudioDataCh = make(chan []byte)
	go func() {
		ctx := context.Background()
		err := t.speechToText(ctx, inAudioConf, t.inAudioDataCh)
		if err != nil {
			toTextNotifyCb(conn, fmt.Errorf("can not convert audio to text: %w", err))
		}
	}
}

func (t *Translator) ToTextContent(dataBytes []byte) {
	if !t.progressInAudio {
		return
	}
	t.inAudioDataCh <- dataBytes
}

func (t *Translator) ToTextContentEnd() {
	if !t.progressInAudio {
		return
	}
	close(t.inAudioDataCh)
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
            SourceLanguageCode: translateConf.srcLang,
            TargetLanguageCode: translateConf.dstLang,
            MimeType:           "text/plain",
            Contents:           t.srcText,
        }
        resp, err := client.TranslateText(ctx, req)
            if err != nil {
                return fmt.Errorf(" can not translate text: %v", err)
        }
        for _, translation := range resp.GetTranslations() {
		t.dstText = append(t.dstText. translation.GetTranslatedText())
        }
}

func  (t *Translator) textToSpeech(ctx context.Context, transConf *message.TransConf) {
        client, err := texttospeech.NewClient(ctx)
        if err != nil {
		return fmt.Errorf("can not create text to speech client: %v", err)
        }
        defer client.Close()
        // Perform the text-to-speech request on the text input with the selected
        // voice parameters and audio file type.
        req := texttospeechpb.SynthesizeSpeechRequest{
                // Set the text input to be synthesized.
                Input: &texttospeechpb.SynthesisInput{
                        InputSource: &texttospeechpb.SynthesisInput_Text{Text: strings.Join(t.dstText, "\n")},
                },
                // Build the voice request, select the language code ("en-US") and the SSML
                // voice gender ("neutral").
                Voice: &texttospeechpb.VoiceSelectionParams{
                        LanguageCode: transConf.DstLang,
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL, // XXXXXXXxx model map
                },
                // Select the type of audio file you want returned.
                AudioConfig: &texttospeechpb.AudioConfig{
                        AudioEncoding: texttospeechpb.AudioEncoding_OGG_OPUS,
                },
        }
        resp, err := client.SynthesizeSpeech(ctx, &req)
        if err != nil {
		return fmt.Errorf("can not synthesize speech: %v", err)
        }
	return resp.AudioContent, message.EncodingOggOpus, nil
}

func (t *Translator) Translate(transConf *message.TransConf) ([]byte, string, err) {
	defer func() {
		t.srcText = t.srcText[:0]
		t.dstText = t.dstText[:0]
	}()
	ctx := context.Background()
	err := t.translateText(ctx, transConf)
	if err != nil {
		return  []byte{}, "", fmt.Errorf("can not translate: %w", err)
	}
	outAudioDataBytes, outAudioEncoding, err = t.textToSpeech(ctx, transConf)
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
	return &Translator{
		opts:            baseOpts,
		projectId:       projectId,
		progressInAudio: false,
		inAudioDataCh:   nil,
		srcText:         make([]string, 0, 10),
		dstText:         make([]string, 0, 10),
	}
}


