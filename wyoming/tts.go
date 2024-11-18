package wyoming

import (
	"errors"
	"io"
	"os"

	"github.com/john-pettigrew/wyoming-cli/utils"
)

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

// TTSSupported returns true if TTS is supported by a Wyoming server.
func (w *WyomingConnection) TTSSupported() bool {
	return len(w.VoiceServices.TTS) > 0
}

// SynthesizeAudio sends a "synthesize" command with voiceData options to a Wyoming server and writes the audio response
// to writer. SynthesizeAudio returns a WyomingAudioData describing the audio data or an error.
func (w *WyomingConnection) SynthesizeAudio(text string, voiceData SynthesizeVoiceData, writer io.Writer) (WyomingAudioData, error) {
	if !w.TTSSupported() {
		return WyomingAudioData{}, errors.New("server does not appear to support TTS")
	}

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

// SynthesizeAudioToStdout sends a "synthesize" command with voiceData options to a Wyoming server and
// writes the audio response to Stdout.
func (w *WyomingConnection) SynthesizeAudioToStdout(text string, voiceData SynthesizeVoiceData) error {
	var writer io.Writer = os.Stdout

	// generate audio
	_, err := w.SynthesizeAudio(text, voiceData, writer)
	if err != nil {
		return err
	}
	return nil
}

// SynthesizeAudioToWAVFile sends a "synthesize" command with voiceData options to a Wyoming server and
// creates a new WAV audio file located at WAVFilePath with the audio received.
func (w *WyomingConnection) SynthesizeAudioToWAVFile(text string, voiceData SynthesizeVoiceData, WAVFilePath string) error {
	var tempFile *os.File
	tempFile, err := os.CreateTemp("", "wyoming-audio")
	if err != nil {
		return err
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	// generate audio
	audioData, err := w.SynthesizeAudio(text, voiceData, tempFile)
	if err != nil {
		return err
	}

	// convert audio
	err = utils.ConvertPCMAudioFileToWAVFile(WAVFilePath, tempFile.Name(), int32(audioData.Rate), int16(audioData.Channels), int16(audioData.Width*8))
	if err != nil {
		return err
	}

	return nil
}
