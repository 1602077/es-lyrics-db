package audio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	fluentffmpeg "github.com/modfy/fluent-ffmpeg"
)

type FfmpegConfig struct {
	OutputFormat string `json:"outputFormat"`
	SampleRate   int    `json:"sampleRate"`
	NumChannels  int    `json:"numChannels"`
	BitRate      int    `json:"bitRate"`
}

func Process(in string, config FfmpegConfig) (*Metadata, error) {
	buf := &bytes.Buffer{}

	fne := in[strings.LastIndex(in, "/")+1:]       // filename w/  extension
	fn := fne[0 : len(fne)-len(filepath.Ext(fne))] // filename w/o extension
	out := fmt.Sprintf("../data/proccessed/%v.%v", fn, config.OutputFormat)

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

	m, err := ProbeMetadata(in)
	if err != nil {
		return &Metadata{}, err
	}
	return m, nil
}

func ProcessBatch(in []string, config FfmpegConfig) {
	// TODO (Jack, 21/06/2022): Switch to output chan of Metadata
	done := make(chan struct{})
	for _, s := range in {
		go func(s string, config FfmpegConfig) {
			Process(s, config)
			done <- struct{}{}
		}(s, config)
	}
	<-done
}

type Metadata struct {
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Title    string `json:"title"`
	Track    string `json:"track"`
	Duration string `json:"duration"`
	Filename string `json:"filename"`
}

func ProbeMetadata(in string) (*Metadata, error) {
	d, err := fluentffmpeg.Probe(in)
	if err != nil {
		return &Metadata{}, err
	}

	// Marshal map[string]interface{} into []byte
	dd, _ := json.Marshal(d)
	if err != nil {
		return &Metadata{}, err
	}

	j, err := jsonvalue.Unmarshal(dd)
	if err != nil {
		return &Metadata{}, err
	}

	// Parse Metadata from unmarshalled json
	artist, err := j.GetString("format", "tags", "artist")
	if err != nil {
		log.Print(err)
	}
	album, err := j.GetString("format", "tags", "album")
	if err != nil {
		log.Print(err)
	}
	title, err := j.GetString("format", "tags", "title")
	if err != nil {
		log.Print(err)
	}
	track, err := j.GetString("format", "tags", "track")
	if err != nil {
		log.Print(err)
	}
	duration, err := j.GetString("format", "duration")
	if err != nil {
		log.Print(err)
	}

	return &Metadata{
		Artist:   artist,
		Album:    album,
		Title:    title,
		Track:    track,
		Duration: duration,
	}, nil
}

// prettyPrints a map[string]interfaces for use in debugging.
func prettyPrint(v map[string]interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}
