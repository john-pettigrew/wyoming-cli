package wyoming

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
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

func transcribeAudioGroupsWorker(audioData WyomingAudioData, serverAddr, modelName, language string, audioEventChan <-chan utils.AudioEvent, resultsChan chan<- Transcription, errorChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	for audioEvent := range audioEventChan {
		w, err := Connect(serverAddr)
		if err != nil {
			errorChan <- err
			return
		}

		defer w.Disconnect()
		text, err := w.TranscribeAudio(&audioEvent.SoundBuff, audioData, modelName, language)
		if err != nil {
			errorChan <- err
			return
		}
		resultsChan <- Transcription{Text: text, Start: audioEvent.Start, End: audioEvent.End}
	}
}

// TranscribeAudioGroups transcribes the audio data from reader and sends the results, containing the
// transcriptions with the start and end times, to resultsChan as they are generated. Errors are sent to errorsChan.
// "workersCount" defines the number of transcription requests that are running at once. TranscribeAudioGroups
// closes resultsChan and returns once an error occurs when reading from reader.
func TranscribeAudioGroups(reader io.Reader, audioData WyomingAudioData, serverAddr, modelName, language string, workersCount, audioWindowMS, minSoundDuration, minSilenceDuration int, soundThreshold, silenceThreshold int32, resultsChan chan<- Transcription, errorsChan chan<- error) {
	audioEventChan := make(chan utils.AudioEvent, workersCount)
	wg := sync.WaitGroup{}

	for i := 0; i < workersCount; i += 1 {
		wg.Add(1)
		go transcribeAudioGroupsWorker(audioData, serverAddr, modelName, language, audioEventChan, resultsChan, errorsChan, &wg)
	}

	currentTimeOffsetMS := 0
	for {
		audioEvent, err := utils.DetectNextAudioGroup16Bit(reader, audioData.Rate, audioData.Channels, audioWindowMS, currentTimeOffsetMS, soundThreshold, silenceThreshold, minSoundDuration, minSilenceDuration)
		if err != nil {
			errorsChan <- err
			break
		}

		currentTimeOffsetMS = int(audioEvent.End.Milliseconds())

		audioEventChan <- audioEvent
	}

	close(audioEventChan)
	wg.Wait()
	close(resultsChan)
	return
}

// TranscribeAllAudioGroups transcribes the audio data from reader and returns a slice
// containing the transcriptions with the start and end times. "workersCount" defines
// the number of transcription requests that are running at once. EOF and ErrUnexpectedEOF
// errors are ignored.
func TranscribeAllAudioGroups(reader io.Reader, audioData WyomingAudioData, serverAddr, modelName, language string, workersCount, audioWindowMS, minSoundDuration, minSilenceDuration int, soundThreshold, silenceThreshold int32) ([]Transcription, error) {
	resultsChan := make(chan Transcription)
	errorsChan := make(chan error)
	var transcriptions []Transcription

	go TranscribeAudioGroups(reader, audioData, serverAddr, modelName, language, workersCount, audioWindowMS, minSoundDuration, minSilenceDuration, soundThreshold, silenceThreshold, resultsChan, errorsChan)

	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				return transcriptions, nil
			}

			transcriptions = append(transcriptions, result)
		case err := <-errorsChan:
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, err
			}
		}
	}
}

// TranscribeAllAudioGroupsFromFile transcribes the audio data from a WAV file located at filePath and returns a slice
// containing the transcriptions with the start and end times.
func TranscribeAllAudioGroupsFromFile(filePath, modelName, language, serverAddr string, audioWindowMS, minSoundDuration, minSilenceDuration, workerCount int, soundThreshold, silenceThreshold int32) ([]Transcription, error) {
	WAVFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer WAVFile.Close()

	rate, channels, bitsPerSample, PCMAudioByteOffset, err := utils.ReadAudioInfoFromWAVFile(WAVFile)
	if err != nil {
		return nil, err
	}

	var width int = int(bitsPerSample / 8)

	if width != 2 {
		return nil, errors.New("only 16-bit audio is supported")
	}

	_, err = WAVFile.Seek(PCMAudioByteOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	transcriptions, err := TranscribeAllAudioGroups(WAVFile, WyomingAudioData{Rate: int(rate), Channels: int(channels), Width: width}, serverAddr, modelName, language, workerCount, audioWindowMS, minSoundDuration, minSilenceDuration, soundThreshold, silenceThreshold)
	if err != nil {
		return nil, err
	}
	return transcriptions, nil
}
