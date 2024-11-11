package utils

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
)

func ConvertPCMAudioToWav(PCMFilePath, outputWavFilePath string, rate, channels int) error {
	// Validate PCM file
	_, err := os.Stat(PCMFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("missing PCM file")
		}
		return err
	}

	// Validate result file
	_, err = os.Stat(outputWavFilePath)
	if err == nil {
		return errors.New("output file already exists")
	} else if !os.IsNotExist(err) {
		return err
	}

	// convert
	cmd := exec.Command(
		"ffmpeg",
		"-f", "s16le",
		"-ar", strconv.Itoa(rate),
		"-ac", strconv.Itoa(channels),
		"-i", PCMFilePath,
		outputWavFilePath,
	)

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
