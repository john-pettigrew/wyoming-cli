# Wyoming CLI
A client for Wyoming servers written in Go.

> Note: This project is currently in alpha!

Current Support:
- TTS

## Examples

### TTS
- output to file:
```
go run commands/tts.go -addr 'localhost:10200' -text 'Hello world' --output_file './hello.wav'
```

- stream raw audio output to speaker:
```
go run commands/tts.go -addr 'localhost:10200' -text 'Hello world' --output-raw | aplay -r 22050 -f S16_LE -t raw -
```
