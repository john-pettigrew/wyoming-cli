package utils

import (
	"bytes"
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

// ReadAudioInfoFromWAVFile returns the audio rate, number of channels, and bitsPerSample read from WAVFile.
func ReadAudioInfoFromWAVFile(WAVFile *os.File) (int32, int16, int16, error) {
	type WAVHeaderField struct {
		Value         []byte
		RequiredValue []byte
		Offset        int64
	}

	WAVHeaderFields := map[string]WAVHeaderField{
		"chunkID": {
			Value:         make([]byte, 4),
			RequiredValue: []byte("RIFF"),
			Offset:        0,
		},
		"chunkSize": {
			Value:  make([]byte, 4),
			Offset: 4,
		},
		"format": {
			Value:         make([]byte, 4),
			RequiredValue: []byte("WAVE"),
			Offset:        8,
		},
		"subchunk1ID": {
			Value:         make([]byte, 4),
			RequiredValue: []byte("fmt "),
			Offset:        12,
		},
		"subchunk1Size": {
			Value:  make([]byte, 4),
			Offset: 16,
		},
		"audioFormat": {
			Value:         make([]byte, 2),
			RequiredValue: []byte{0x01, 0x00},
			Offset:        20,
		},
		"channels": {
			Value:  make([]byte, 2),
			Offset: 22,
		},
		"sampleRate": {
			Value:  make([]byte, 4),
			Offset: 24,
		},
		"byteRate": {
			Value:  make([]byte, 4),
			Offset: 28,
		},
		"blockAlign": {
			Value:  make([]byte, 2),
			Offset: 32,
		},
		"bitsPerSample": {
			Value:  make([]byte, 2),
			Offset: 34,
		},
		"subchunk2ID": {
			Value:         make([]byte, 4),
			RequiredValue: []byte("data"),
			Offset:        36,
		},
	}

	for _, field := range WAVHeaderFields {
		_, err := WAVFile.Seek(field.Offset, io.SeekStart)
		if err != nil {
			return 0, 0, 0, err
		}

		err = binary.Read(WAVFile, binary.LittleEndian, &field.Value)
		if err != nil {
			return 0, 0, 0, err
		}

		if field.RequiredValue != nil {
			if !bytes.Equal(field.Value, field.RequiredValue) {
				return 0, 0, 0, errors.New("invalid WAV header")
			}
		}
	}
	var rate int32
	var channels int16
	var bitsPerSample int16

	rateBuff := bytes.NewBuffer(WAVHeaderFields["sampleRate"].Value)
	channelsBuff := bytes.NewBuffer(WAVHeaderFields["channels"].Value)
	bitsPerSampleBuff := bytes.NewBuffer(WAVHeaderFields["bitsPerSample"].Value)

	err := binary.Read(rateBuff, binary.LittleEndian, &rate)
	if err != nil {
		return 0, 0, 0, err
	}

	err = binary.Read(channelsBuff, binary.LittleEndian, &channels)
	if err != nil {
		return 0, 0, 0, err
	}

	err = binary.Read(bitsPerSampleBuff, binary.LittleEndian, &bitsPerSample)
	if err != nil {
		return 0, 0, 0, err
	}

	return rate, channels, bitsPerSample, nil
}
