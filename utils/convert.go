package utils

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// ConvertPCMAudioFileToWAVFile converts the PCM audio file located at PCMFilePath to a new WAV file located at WAVFilePath.
func ConvertPCMAudioFileToWAVFile(WAVFilePath, PCMFilePath string, rate int32, channels int16, bitsPerSample int16) error {
	_, err := os.Stat(WAVFilePath)
	if err == nil {
		return errors.New("output file already exists")
	} else if !os.IsNotExist(err) {
		return err
	}

	outputFile, err := os.Create(WAVFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	PCMFile, err := os.Open(PCMFilePath)
	if err != nil {
		return err
	}
	defer PCMFile.Close()

	PCMFileStat, err := PCMFile.Stat()
	if err != nil {
		return err
	}

	err = ConvertPCMAudioToWAV(outputFile, PCMFile, int(PCMFileStat.Size()), rate, channels, bitsPerSample)
	if err != nil {
		return err
	}

	return nil
}

// ConvertPCMAudioToWAV writes necessary WAV file header data and PCM audio data from PCMReader to WAVWriter.
func ConvertPCMAudioToWAV(WAVWriter io.Writer, PCMReader io.Reader, PCMDataLength int, rate int32, channels int16, bitsPerSample int16) error {
	// write WAV header
	var blockAlign int16 = channels * (bitsPerSample / 8)
	var byteRate int32 = rate * int32(blockAlign)

	WAVHeaderFields := []any{
		// RIFF
		[]byte("RIFF"),            // Chunk ID
		int32(36 + PCMDataLength), // Chunk Size
		[]byte("WAVE"),            // Format

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
		[]byte("data"),       // Subchunk2 ID
		int32(PCMDataLength), // Subchunk2 Size
	}

	for _, field := range WAVHeaderFields {
		err := binary.Write(WAVWriter, binary.LittleEndian, field)
		if err != nil {
			return err
		}
	}

	// write audio data
	_, err := io.Copy(WAVWriter, PCMReader)
	if err != nil {
		return err
	}

	return nil
}
