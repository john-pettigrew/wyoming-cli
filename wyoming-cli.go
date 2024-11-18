package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/john-pettigrew/wyoming-cli/commands"
)

func main() {
	var err error

	switch os.Args[1] {
	case "tts":
		err = commands.TTS()
	default:
		err = errors.New("unknown command")
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
