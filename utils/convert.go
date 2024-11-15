package utils

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

func ConvertPCMAudioToWav(PCMFilePath, outputWavFilePath string, rate int32, channels int16, bitsPerSample int16) error {
	// Validate PCM file
	PCMFileStat, err := os.Stat(PCMFilePath)
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

	// open files
	outputFile, err := os.Create(outputWavFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	PCMFile, err := os.Open(PCMFilePath)
	if err != nil {
		return err
	}
	defer PCMFile.Close()

	// write WAV header
	var blockAlign int16 = channels * (bitsPerSample / 8)
	var byteRate int32 = rate * int32(blockAlign)

	WAVHeaderFields := []any{
		// RIFF
		[]byte("RIFF"),                 // Chunk ID
		int32(36 + PCMFileStat.Size()), // Chunk Size
		[]byte("WAVE"),                 // Format

		// fmt
		[]byte("fmt "),       // Subchunk1 ID
		int32(16),            // Subchunk1 Size
		int16(1),             // AudioFormat (PCM)
		int16(channels),      // Num Channels
		int32(rate),          // Sample Rate
		int32(byteRate),      // Byte Rate
		int16(blockAlign),    // Block Align
		int16(bitsPerSample), // Bits Per Sample

		// data
		[]byte("data"),            // Subchunk2 ID
		int32(PCMFileStat.Size()), // Subchunk2 Size
	}

	for _, field := range WAVHeaderFields {
		err = binary.Write(outputFile, binary.LittleEndian, field)
		if err != nil {
			return err
		}
	}

	// write audio data
	_, err = io.Copy(outputFile, PCMFile)
	if err != nil {
		return err
	}

	return nil
}
