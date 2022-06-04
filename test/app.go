package main

import (
    "context"
    "fmt"

    translate "cloud.google.com/go/translate/apiv3"
    translatepb "google.golang.org/genproto/googleapis/cloud/translate/v3"
)

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
	 //translateText("clipper-255807", "en-us", "ja", []string{"hello, world!", "I am hungry."})
	 translateText("clipper-255807", "ja-JP", "en", []string{"おはようございます今日はいい天気ですねそれにしてもお腹が空きました"})
}
