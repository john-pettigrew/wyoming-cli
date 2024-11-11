package wyoming

var SynthesizeMessageType string = "synthesize"

type SynthesizeVoiceData struct {
	Name     string `json:"name,omitempty"`
	Language string `json:"language,omitempty"`
	Speaker  string `json:"speaker,omitempty"`
}

type SynthesizeData struct {
	Text  string              `json:"text"`
	Voice SynthesizeVoiceData `json:"voice,omitempty"`
}

type WyomingVoiceServicesTTSData struct {
	Name      string   `json:"name"`
	Languages []string `json:"languages"`
	Voices    []struct {
		Name        string             `json:"name"`
		Attribution WyomingAttribution `json:"attribution"`
	} `json:"voices"`
	Speakers []struct {
		Name string `json:"name,omitempty"`
	} `json:"speakers"`
	Attribution WyomingAttribution `json:"attribution"`
	Installed   bool               `json:"installed"`
	Description string             `json:"description,omitempty"`
	Version     string             `json:"version,omitempty"`
}

func (w *WyomingConnection) SythesizeAudio(text string, voiceData SynthesizeVoiceData, outputRawData bool, outputFilePath string) error {
	err := w.SendMessage(WyomingMessage{Type: SynthesizeMessageType, Data: SynthesizeData{
		Text:  text,
		Voice: voiceData,
	}})
	if err != nil {
		return err
	}

	err = w.ReceiveAudio(outputRawData, outputFilePath)
	if err != nil {
		return err
	}

	return nil
}
