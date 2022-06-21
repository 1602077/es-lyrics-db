package main

import (
	"fmt"

	"github.com/1602077/es-lyrics-db/pkg/audio"
)

func main() {

	ac := audio.FfmpegConfig{
		OutputFormat: "wav",
		SampleRate:   44100,
		NumChannels:  2,
		BitRate:      160,
	}

	fn := "01 Smoke Signals"
	in := fmt.Sprintf("../data/%v.mp3", fn)

	audio.ProcessBatch([]string{in}, ac)
}
