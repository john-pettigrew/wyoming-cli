package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/john-pettigrew/wyoming-cli/wyoming"
)

func validateInputs(text, serverAddr, outputFilePath string, outputRawData bool) error {
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

func main() {
	text := flag.String("text", "", "text to be spoken")
	serverAddr := flag.String("addr", "localhost:10200", "address and port for tts Wyoming server")
	outputFilePath := flag.String("output_file", "", "output file path")
	outputRawData := flag.Bool("output-raw", false, "stream audio data to stdout")

	voiceName := flag.String("voice-name", "", "voice name")

	flag.Parse()

	if err := validateInputs(*text, *serverAddr, *outputFilePath, *outputRawData); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// connect to server
	wyomingConn, err := wyoming.Connect(*serverAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer wyomingConn.Disconnect()

	voiceServices, err := wyomingConn.GetAvailableServices()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(voiceServices.TTS) == 0 {
		err = errors.New("server does not appear to support TTS")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// generate audio
	err = wyomingConn.SythesizeAudio(*text, wyoming.SynthesizeVoiceData{Name: *voiceName}, *outputRawData, *outputFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
