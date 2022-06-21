package audio

import (
	"encoding/json"
	"fmt"
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

// Process uses ffmpeg to convert input path's file according to the specified
// ffmpeg config provided.
func Process(path string, config FfmpegConfig) (*Metadata, error) {
	fne := path[strings.LastIndex(path, "/")+1:]
	fn := fne[0 : len(fne)-len(filepath.Ext(fne))]
	out := fmt.Sprintf("../data/proccessed/%v.%v", fn, config.OutputFormat)

	fluentffmpeg.NewCommand("").
		InputPath(path).
		AudioChannels(config.NumChannels).
		AudioRate(config.SampleRate).
		AudioBitRate(config.BitRate).
		OutputFormat(config.OutputFormat).
		OutputPath(out).
		Overwrite(true).
		Run()

	m, err := ProbeMetadata(path)
	if err != nil {
		return &Metadata{}, err
	}
	return m, nil
}

// ProcessBatch concurrently runs Process for a specified slice of input paths.
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

// ProbeMetadata uses a ffmpeg probe to extract music metadata from input path.
func ProbeMetadata(path string) (*Metadata, error) {
	d, err := fluentffmpeg.Probe(path)
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

	// Parse Metadata from un-marshalled json
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
