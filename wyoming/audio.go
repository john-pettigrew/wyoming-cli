package wyoming

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"

	"github.com/john-pettigrew/wyoming-cli/utils"
)

var AudioStartMessageType string = "audio-start"
var AudioChunkMessageType string = "audio-chunk"
var AudioStopMessageType string = "audio-stop"

type WyomingAudioData struct {
	Rate      int    `json:"rate"`
	Width     int    `json:"width"`
	Channels  int    `json:"channels"`
	Timestamp string `json:"timestamp"`
}

func (w *WyomingConnection) ReceiveAudio(outputRawData bool, outputPath string) error {
	var audioData WyomingAudioData
	reader := bufio.NewReader(w.Conn)

	tempFile, err := os.CreateTemp("", "wyoming-audio")
	if err != nil {
		return err
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	if outputPath == "" && !outputRawData {
		return errors.New("missing output path")
	}

	for {
		res, err := w.ReceiveMessageUsingReader(reader)
		if err != nil {
			return err
		}

		if res.Message.Type == AudioChunkMessageType {
			if audioData.Rate == 0 && len(res.Data) > 0 {
				err = json.Unmarshal(res.Data, &audioData)
				if err != nil {
					return err
				}
			}

			if len(res.Payload) > 0 {
				if outputRawData {
					err = binary.Write(os.Stdout, binary.LittleEndian, res.Payload)
					if err != nil {
						return err
					}
				} else {
					_, err = tempFile.Write(res.Payload)
					if err != nil {
						return err
					}
				}
			}
		}

		if res.Message.Type == AudioStopMessageType {
			break
		}
	}

	if !outputRawData {
		err = utils.ConvertPCMAudioToWav(tempFile.Name(), outputPath, audioData.Rate, audioData.Channels)
		if err != nil {
			return err
		}
	}

	return nil
}
