package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"time"
)

var DETECT_NOISE_MODE int = 0
var DETECT_SILENCE_MODE int = 1

type AudioEvent struct {
	Start     time.Duration
	End       time.Duration
	SoundBuff bytes.Buffer
}

// DetectAudioEvent16Bits reads from reader and detects if an audio event occurred during one audioWindowMS duration. An event is
// detected by subtracting the smallest audio value read from the largest and comparing the result to audioThreshold. If mode is set
// to DETECT_NOISE_MODE then the result must be greater than audioThreshold. If mode is set to DETECT_SILENCE_MODE then it must be less.
func DetectAudioEvent16Bits(reader io.Reader, mode, rate, channels, audioWindowMS int, audioThreshold int32) (bool, error) {
	if mode != DETECT_NOISE_MODE && mode != DETECT_SILENCE_MODE {
		return false, errors.New("invalid mode")
	}
	windowSize := int((float64(rate) / (1000 / float64(audioWindowMS))) * float64(channels))
	if windowSize <= 0 {
		return false, errors.New("invalid window size")
	}

	var highestValue int32 = 0
	var lowestValue int32 = 0
	var currentValue int16 = 0
	var currentValue32 int32 = 0

	for i := 0; i < windowSize; i += 1 {
		err := binary.Read(reader, binary.LittleEndian, &currentValue)
		if err != nil {
			return false, err
		}

		currentValue32 = int32(currentValue) + math.MaxInt16

		if currentValue32 > highestValue {
			highestValue = currentValue32
		}
		if currentValue32 < lowestValue {
			lowestValue = currentValue32
		}
	}

	if mode == DETECT_NOISE_MODE && highestValue-lowestValue > audioThreshold {
		return true, nil
	} else if mode == DETECT_SILENCE_MODE && highestValue-lowestValue < audioThreshold {
		return true, nil
	}

	return false, nil
}

// DetectNextAudioGroup16Bit reads from reader and detects the next segment of audio. The AudioEvent returned contains the start
// and end time offset by offsetMS and a buffer containing the audio data.
func DetectNextAudioGroup16Bit(reader io.Reader, rate, channels, audioWindowMS, offsetMS int, soundThreshold, silenceThreshold int32, soundDurationMS, silenceDurationMS int) (AudioEvent, error) {
	var startTimeMS int = offsetMS
	var endTimeMS int

	soundEvents := 0
	silenceEvents := 0
	minSoundEvents := soundDurationMS / audioWindowMS
	minSilenceEvents := silenceDurationMS / audioWindowMS

	soundBuff := bytes.Buffer{}
	teeReader := io.TeeReader(reader, &soundBuff)
	for {
		hasSound, err := DetectAudioEvent16Bits(teeReader, DETECT_NOISE_MODE, rate, channels, audioWindowMS, soundThreshold)
		if err != nil {
			return AudioEvent{}, err
		}

		if hasSound {
			soundEvents += 1
		} else {
			soundBuff.Reset()
			soundEvents = 0
		}

		if soundEvents >= minSoundEvents {
			break
		}

		startTimeMS += audioWindowMS
	}

	endTimeMS = startTimeMS

	for {
		hasSilence, err := DetectAudioEvent16Bits(teeReader, DETECT_SILENCE_MODE, rate, channels, audioWindowMS, silenceThreshold)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return AudioEvent{}, err
		}

		if hasSilence {
			silenceEvents += 1
		} else {
			silenceEvents = 0
		}

		if silenceEvents >= minSilenceEvents {
			break
		}
		endTimeMS += audioWindowMS
	}
	return AudioEvent{Start: time.Millisecond * time.Duration(startTimeMS), End: time.Millisecond * time.Duration(endTimeMS), SoundBuff: soundBuff}, nil
}
