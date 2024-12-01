package wyoming

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

var AudioStartMessageType string = "audio-start"
var AudioChunkMessageType string = "audio-chunk"
var AudioStopMessageType string = "audio-stop"

type WyomingAudioData struct {
	Rate      int    `json:"rate"`
	Width     int    `json:"width"`
	Channels  int    `json:"channels"`
	Timestamp string `json:"timestamp,omitempty"`
}

// ReceiveAudio writes audio data to writer from "audio-chunk" messages as they are received. ReceiveAudio
// stops listening for data once an "audio-stop" message is sent. ReceiveAudio returns a WyomingAudioData describing
// the audio data or an error.
func (w *WyomingConnection) ReceiveAudio(writer io.Writer) (WyomingAudioData, error) {
	var audioData WyomingAudioData
	reader := bufio.NewReader(w.Conn)

	for {
		res, err := w.ReceiveMessageUsingReader(reader)
		if err != nil {
			return WyomingAudioData{}, err
		}

		if res.Message.Type == AudioChunkMessageType {
			if audioData.Rate == 0 && len(res.Data) > 0 {
				err = json.Unmarshal(res.Data, &audioData)
				if err != nil {
					return WyomingAudioData{}, err
				}
			}

			if len(res.Payload) > 0 {
				err = binary.Write(writer, binary.LittleEndian, res.Payload)
				if err != nil {
					return WyomingAudioData{}, err
				}
			}
		}

		if res.Message.Type == AudioStopMessageType {
			break
		}
	}

	return audioData, nil
}

// SendAudio sends audio data from reader to the Wyoming server until an EOF error is detected.
func (w *WyomingConnection) SendAudio(reader io.Reader, audioData WyomingAudioData) error {
	buf := make([]byte, 1024)
	err := w.SendMessageContainer(
		WyomingMessageContainer{
			Message: WyomingMessage{Type: AudioStartMessageType, Data: audioData},
		},
	)

	if err != nil {
		return err
	}

	for {
		_, err := reader.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		err = w.SendMessageContainer(
			WyomingMessageContainer{
				Message: WyomingMessage{Type: AudioChunkMessageType, Data: audioData},
				Payload: buf,
			},
		)
		if err != nil {
			return err
		}
	}

	err = w.SendMessageContainer(
		WyomingMessageContainer{
			Message: WyomingMessage{
				Type: AudioStopMessageType,
				Data: audioData,
			},
		},
	)

	if err != nil {
		return err
	}

	return nil
}
