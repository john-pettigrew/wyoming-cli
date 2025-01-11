package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/john-pettigrew/wyoming-cli/wyoming"
)

func validateInputsASR(serverAddr, inputFilePath string, inputRawData bool, inputRawDataRate, inputRawDataChannels, audioWindowMS int, soundThreshold, silenceThreshold int32, minSoundDuration, minSilenceDuration int) error {
	if serverAddr == "" {
		return errors.New("missing server address")
	}

	if audioWindowMS <= 0 {
		return errors.New("audio-window-ms must be greater than 0")
	}
	if soundThreshold <= 0 {
		return errors.New("sound-threshold must be greater than 0")
	}
	if silenceThreshold <= 0 {
		return errors.New("silence-threshold must be greater than 0")
	}
	if minSoundDuration <= 0 {
		return errors.New("min-sound-duration-ms must be greater than 0")
	}
	if minSoundDuration%audioWindowMS != 0 {
		return errors.New("min-sound-duration-ms must be divisible by audio-window-ms")
	}
	if minSilenceDuration <= 0 {
		return errors.New("min-silence-duration-ms must be greater than 0")
	}
	if minSilenceDuration%audioWindowMS != 0 {
		return errors.New("min-silence-duration-ms must be divisible by audio-window-ms")
	}

	if inputRawData {
		if inputRawDataRate <= 0 {
			return errors.New("input-raw-rate must be greater than 0")
		}
		if inputRawDataChannels <= 0 {
			return errors.New("input-raw-channels must be greater than 0")
		}
	} else {
		if inputFilePath == "" {
			return errors.New("missing input file path")
		}

		if len(inputFilePath) < 4 || strings.ToLower(inputFilePath[len(inputFilePath)-4:]) != ".wav" {
			return errors.New("input_file must be a WAV audio file")
		}

		_, err := os.Stat(inputFilePath)
		if err != nil {
			return err
		}

	}

	return nil
}

func parseAndValidateFlagsASR(currentFlag *flag.FlagSet) (string, string, string, string, bool, int, int, int, int32, int32, int, int, int, error) {
	serverAddr := currentFlag.String("addr", "localhost:10300", "address and port for asr Wyoming server")
	inputFilePath := currentFlag.String("input_file", "", "input WAV file path")
	modelName := currentFlag.String("model-name", "", "name of model")
	language := currentFlag.String("language", "", "language")

	inputRawData := currentFlag.Bool("input-raw", false, "listen for audio data from stdin and output results to stdout in a loop")
	inputRawDataRate := currentFlag.Int("input-raw-rate", 22050, "audio rate from stdin")
	inputRawDataChannels := currentFlag.Int("input-raw-channels", 1, "number of audio channels from stdin")

	numWorkers := currentFlag.Int("num-workers", 3, "number of workers")
	audioWindowMS := currentFlag.Int("audio-window-ms", 100, "window size in MS to use for detecting sound")
	soundThreshold := currentFlag.Int("sound-threshold", 20000, "level of noise for a sound event")
	silenceThreshold := currentFlag.Int("silence-threshold", 2000, "level of noise for a silence event")
	minSoundDuration := currentFlag.Int("min-sound-duration-ms", 100, "minimum length of a sound event")
	minSilenceDuration := currentFlag.Int("min-silence-duration-ms", 100, "minimum length of a silence event")

	currentFlag.Parse(os.Args[2:])

	if err := validateInputsASR(
		*serverAddr,
		*inputFilePath,
		*inputRawData,
		*inputRawDataRate,
		*inputRawDataChannels,
		*audioWindowMS,
		int32(*soundThreshold),
		int32(*silenceThreshold),
		*minSoundDuration,
		*minSilenceDuration,
	); err != nil {
		return "", "", "", "", false, 0, 0, 0, 0, 0, 0, 0, 0, err
	}

	return *serverAddr, *inputFilePath, *modelName, *language, *inputRawData, *inputRawDataRate, *inputRawDataChannels, *audioWindowMS, int32(*soundThreshold), int32(*silenceThreshold), *minSoundDuration, *minSilenceDuration, *numWorkers, nil
}

func ASR() error {
	currentFlag := flag.NewFlagSet("asr", flag.ExitOnError)

	serverAddr, inputFilePath, modelName, language, inputRawData, inputRawDataRate, inputRawDataChannels, audioWindowMS, soundThreshold, silenceThreshold, minSoundDuration, minSilenceDuration, numWorkers, err := parseAndValidateFlagsASR(currentFlag)
	if err != nil {
		return err
	}

	if !inputRawData {
		transcriptions, err := wyoming.TranscribeAllAudioGroupsFromFile(inputFilePath, modelName, language, serverAddr, audioWindowMS, minSoundDuration, minSilenceDuration, numWorkers, soundThreshold, silenceThreshold)
		if err != nil {
			return err
		}

		for i, transcription := range transcriptions {
			_, err = fmt.Printf("%d: %f - %f '%s'\n", i, transcription.Start.Seconds(), transcription.End.Seconds(), transcription.Text)
			if err != nil {
				return err
			}
		}
		return nil
	}

	resultsChan := make(chan wyoming.Transcription)
	errorsChan := make(chan error)

	go wyoming.TranscribeAudioGroups(os.Stdin, wyoming.WyomingAudioData{Rate: inputRawDataRate, Width: 2, Channels: inputRawDataChannels}, serverAddr, modelName, language, numWorkers, audioWindowMS, minSoundDuration, minSilenceDuration, soundThreshold, silenceThreshold, resultsChan, errorsChan)

	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				return nil
			}

			fmt.Println(result.Text)
		case err := <-errorsChan:
			return err
		}
	}
}
