# Wyoming CLI
A client for Wyoming servers written in Go.

> Note: This project is currently in alpha!

Supports:
- TTS
- ASR

## Examples

### TTS
- output to file:
```
go run wyoming-cli.go tts -addr 'localhost:10200' -text 'Hello world' --output_file './hello.wav'
```

- stream raw audio output to speaker:
```
go run wyoming-cli.go tts -addr 'localhost:10200' -text 'Hello world' --output-raw | aplay -r 22050 -f S16_LE -t raw -
```

### ASR
- read from WAV file:
```
go run wyoming-cli.go asr --input_file './hello.wav'
```

- read from mic using Stdin:
```
arecord -f S16_LE -r 22050 -c 1 -t raw - | go run wyoming-cli.go asr --input-raw
```