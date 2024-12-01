package wyoming

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/john-pettigrew/wyoming-cli/utils"
)

var TranscribeMessageType string = "transcribe"
var TranscriptMessageType string = "transcript"

type TranscribeData struct {
	Name     string `json:"name,omitempty"`
	Language string `json:"language,omitempty"`
}
type TranscriptionData struct {
	Text string `json:"text"`
}

type WyomingVoiceServicesASRData struct {
	Name        string             `json:"name"`
	Languages   []string           `json:"languages"`
	Attribution WyomingAttribution `json:"attribution"`
	Installed   bool               `json:"installed"`
	Description string             `json:"description,omitempty"`
	Version     string             `json:"version,omitempty"`
}

type Transcription struct {
	Text  string
	Start time.Duration
	End   time.Duration
}

// ASRSupported returns true if ASR is supported by a Wyoming server.
func (w *WyomingConnection) ASRSupported() bool {
	return len(w.VoiceServices.ASR) > 0
}

// TranscribeAudioFromFile transcribes the audio data from a WAV file located at filePath and returns a slice
// containing the transcriptions with the start and end times.
func (w *WyomingConnection) TranscribeAudioFromFile(filePath string, audioWindowMS int, soundThreshold, silenceThreshold int32, minSoundDuration, minSilenceDuration int, modelName, language string) ([]Transcription, error) {
	if !w.ASRSupported() {
		return nil, errors.New("server does not appear to support ASR")
	}

	WAVFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer WAVFile.Close()

	rate, channels, bitsPerSample, err := utils.ReadAudioInfoFromWAVFile(WAVFile)
	if err != nil {
		return nil, err
	}

	var width int = int(bitsPerSample / 8)
	var transcriptions []Transcription
	var PCMAudioByteOffset int64 = 40

	if width != 2 {
		return nil, errors.New("only 16-bit audio is supported")
	}

	_, err = WAVFile.Seek(PCMAudioByteOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	currentTimeOffsetMS := 0
	for {
		text, audioEvent, err := w.TranscribeNextAudio(WAVFile, WyomingAudioData{Rate: int(rate), Channels: int(channels), Width: width}, audioWindowMS, currentTimeOffsetMS, soundThreshold, silenceThreshold, minSoundDuration, minSilenceDuration, modelName, language)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}

		currentTimeOffsetMS = int(audioEvent.End.Milliseconds())

		err = w.Reconnect()
		if err != nil {
			return nil, err
		}

		transcriptions = append(transcriptions, Transcription{Text: text, Start: audioEvent.Start, End: audioEvent.End})
	}

	return transcriptions, nil
}

// TranscribeAudio sends a "transcribe" request to the Wyoming server followed by the audio data from reader and returns the
// result.
func (w *WyomingConnection) TranscribeAudio(reader io.Reader, audioData WyomingAudioData, modelName, language string) (string, error) {
	err := w.SendMessage(WyomingMessage{Type: TranscribeMessageType, Data: TranscribeData{
		Name:     modelName,
		Language: language,
	}})
	if err != nil {
		return "", err
	}
	err = w.SendAudio(reader, audioData)
	if err != nil {
		return "", err
	}

	responseMsg, err := w.ReceiveMessage()
	if err != nil {
		return "", err
	}
	if responseMsg.Message.Type != TranscriptMessageType {
		return "", errors.New("unexpected response message")
	}

	var transcriptionData TranscriptionData
	err = json.Unmarshal(responseMsg.Data, &transcriptionData)
	if err != nil {
		return "", err
	}

	return transcriptionData.Text, nil
}

// TranscribeNextAudio detects the next audio segment from reader and requests a transcription from the Wyoming server.
func (w *WyomingConnection) TranscribeNextAudio(reader io.Reader, audioData WyomingAudioData, audioWindowMS, timeOffsetMS int, soundThreshold, silenceThreshold int32, minSoundDuration, minSilenceDuration int, modelName, language string) (string, utils.AudioEvent, error) {
	audioEvent, err := utils.DetectNextAudioGroup16Bit(reader, audioData.Rate, audioData.Channels, audioWindowMS, timeOffsetMS, soundThreshold, silenceThreshold, minSoundDuration, minSilenceDuration)
	if err != nil {
		return "", utils.AudioEvent{}, err
	}

	text, err := w.TranscribeAudio(&audioEvent.SoundBuff, audioData, modelName, language)
	if err != nil {
		return "", utils.AudioEvent{}, err
	}

	return text, audioEvent, nil
}
