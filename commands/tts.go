package commands

import (
	"errors"
	"flag"
	"os"

	"github.com/john-pettigrew/wyoming-cli/wyoming"
)

func validateInputsTTS(text, serverAddr, outputFilePath string, outputRawData bool) error {
	if text == "" {
		return errors.New("missing text")
	}
	if serverAddr == "" {
		return errors.New("missing server address")
	}
	if !outputRawData {
		if outputFilePath == "" {
			return errors.New("missing output file path")
		}

		_, err := os.Stat(outputFilePath)
		if err == nil {
			return errors.New("output file already exists")
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func parseAndValidateFlagsTTS(currentFlag *flag.FlagSet) (string, string, string, string, bool, error) {
	text := currentFlag.String("text", "", "text to be spoken")
	serverAddr := currentFlag.String("addr", "localhost:10200", "address and port for tts Wyoming server")
	outputFilePath := currentFlag.String("output_file", "", "output file path")
	outputRawData := currentFlag.Bool("output-raw", false, "stream audio data to stdout")

	voiceName := currentFlag.String("voice-name", "", "voice name")

	currentFlag.Parse(os.Args[2:])

	if err := validateInputsTTS(*text, *serverAddr, *outputFilePath, *outputRawData); err != nil {
		return "", "", "", "", false, err
	}

	return *text, *serverAddr, *outputFilePath, *voiceName, *outputRawData, nil
}

func TTS() error {
	currentFlag := flag.NewFlagSet("tts", flag.ExitOnError)

	text, serverAddr, outputFilePath, voiceName, outputRawData, err := parseAndValidateFlagsTTS(currentFlag)
	if err != nil {
		return err
	}

	// connect to server
	wyomingConn, err := wyoming.Connect(serverAddr)
	if err != nil {
		return err
	}
	defer wyomingConn.Disconnect()

	// synthesize audio
	if outputRawData {
		err = wyomingConn.SynthesizeAudioToStdout(text, wyoming.SynthesizeVoiceData{Name: voiceName})
		if err != nil {
			return err
		}

		return nil
	}

	err = wyomingConn.SynthesizeAudioToWAVFile(text, wyoming.SynthesizeVoiceData{Name: voiceName}, outputFilePath)
	if err != nil {
		return err
	}

	return nil
}
