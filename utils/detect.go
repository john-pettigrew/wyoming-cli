package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"time"
)

const DETECT_NOISE_MODE int = 0
const DETECT_SILENCE_MODE int = 1

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
		if currentValue32 < lowestValue || lowestValue == 0 {
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

// DetectAudioEventDuration16Bits reads from reader and detects an audio event defined by detectMode, durationMS, and audioThreshold. DetectAudioEventDuration16Bits
// returns the offset of the start of the event in MS and a buffer containing the audio of the event. If detectMode is set to DETECT_SILENCE_MODE and an EOF error
// is returned while reading from reader then durationMS is ignored and DetectAudioEventDuration16Bits returns a silence event.
func DetectAudioEventDuration16Bits(reader io.Reader, rate, channels, durationMS, audioWindowMS, detectMode int, audioThreshold int32) (int, bytes.Buffer, error) {
	events := 0
	minEvents := durationMS / audioWindowMS

	soundBuff := bytes.Buffer{}
	teeReader := io.TeeReader(reader, &soundBuff)
	startTimeMS := 0
	currentOffsetMS := 0

	for {
		eventDetected, err := DetectAudioEvent16Bits(teeReader, detectMode, rate, channels, audioWindowMS, audioThreshold)
		if err != nil {
			if errors.Is(err, io.EOF) && detectMode == DETECT_SILENCE_MODE {
				if events == 0 {
					startTimeMS = currentOffsetMS
					soundBuff.Reset()
				}
				return startTimeMS, soundBuff, nil
			}

			return 0, bytes.Buffer{}, err
		}

		if eventDetected {
			if events == 0 {
				startTimeMS = currentOffsetMS
			}
			events += 1
		} else {
			soundBuff.Reset()
			events = 0
		}

		if events >= minEvents {
			break
		}

		currentOffsetMS += audioWindowMS
	}

	return startTimeMS, soundBuff, nil
}

// DetectNextAudioGroup16Bit reads from reader and detects the next segment of audio. The AudioEvent returned contains the start
// and end time offset by offsetMS and a buffer containing the audio data.
func DetectNextAudioGroup16Bit(reader io.Reader, rate, channels, audioWindowMS, offsetMS int, soundThreshold, silenceThreshold int32, soundDurationMS, silenceDurationMS int) (AudioEvent, error) {
	soundOffsetMS, soundBuff, err := DetectAudioEventDuration16Bits(reader, rate, channels, soundDurationMS, audioWindowMS, DETECT_NOISE_MODE, soundThreshold)
	if err != nil {
		return AudioEvent{}, err
	}

	startTimeMS := soundOffsetMS + offsetMS

	teeReader := io.TeeReader(reader, &soundBuff)
	silenceOffsetMS, silenceBuff, err := DetectAudioEventDuration16Bits(teeReader, rate, channels, silenceDurationMS, audioWindowMS, DETECT_SILENCE_MODE, silenceThreshold)
	if err != nil {
		return AudioEvent{}, err
	}

	soundBuff.Truncate(soundBuff.Len() - silenceBuff.Len())

	endTimeMS := startTimeMS + soundDurationMS + silenceOffsetMS

	return AudioEvent{Start: time.Millisecond * time.Duration(startTimeMS), End: time.Millisecond * time.Duration(endTimeMS), SoundBuff: soundBuff}, nil
}
