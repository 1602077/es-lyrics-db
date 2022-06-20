package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	fluentffmpeg "github.com/modfy/fluent-ffmpeg"
)

type AudioConfig struct {
	OutputFormat string
	SampleRate   int
	NumChannels  int
	BitRate      int
}

func ProcessAudioFile(in, out string, config AudioConfig) {
	buf := &bytes.Buffer{}

	fluentffmpeg.NewCommand("").
		InputPath(in).
		AudioChannels(config.NumChannels).
		AudioRate(config.SampleRate).
		AudioBitRate(config.BitRate).
		OutputLogs(buf).
		OutputFormat(config.OutputFormat).
		OutputPath(out).
		Overwrite(true).
		Run()

	logs, _ := ioutil.ReadAll(buf)
	fmt.Println(string(logs))
}

func main() {

	ac := AudioConfig{
		OutputFormat: "wav",
		SampleRate:   44100,
		NumChannels:  2,
		BitRate:      160,
	}

	fn := "01 Smoke Signals"
	in := fmt.Sprintf("../testdata/%v.mp3", fn)
	out := fmt.Sprintf("../testdata/output/%v.%v", fn, ac.OutputFormat)

	ProcessAudioFile(in, out, ac)
}
