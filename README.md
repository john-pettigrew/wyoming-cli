# Wyoming CLI 🔊
A command-line client for Wyoming servers written in Go.

> Note: This project is currently in alpha!

## Features:
- TTS (Text to Speech) - output to file or stdout
- ASR (Automatic Speech Recognition) - input from file or stdin

## Installation
To install, run:
```
go install github.com/john-pettigrew/wyoming-cli@latest
```

## Examples

### TTS
- output to file:
```
wyoming-cli tts -addr 'localhost:10200' -text 'Hello world' --output_file './hello.wav'
```

- stream raw audio output to speaker:
```
wyoming-cli tts -addr 'localhost:10200' -text 'Hello world' --output-raw | aplay -r 22050 -f S16_LE -t raw -
```

### ASR
- print text from WAV file audio:
```
wyoming-cli asr --input_file './hello.wav'
```

- print text from mic audio using Stdin:
```
arecord -f S16_LE -r 22050 -c 1 -t raw - | wyoming-cli asr --input-raw
```