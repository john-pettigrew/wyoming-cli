package wyoming

import "io"

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

func (w *WyomingConnection) SynthesizeAudio(text string, voiceData SynthesizeVoiceData, writer io.Writer) (WyomingAudioData, error) {
	err := w.SendMessage(WyomingMessage{Type: SynthesizeMessageType, Data: SynthesizeData{
		Text:  text,
		Voice: voiceData,
	}})
	if err != nil {
		return WyomingAudioData{}, err
	}

	audioData, err := w.ReceiveAudio(writer)
	if err != nil {
		return WyomingAudioData{}, err
	}

	return audioData, nil
}
