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
// ffmpeg config provided saving the processed file into dir.
func Process(path string, dir string, config FfmpegConfig) (*Metadata, error) {
	fne := path[strings.LastIndex(path, "/")+1:]
	fn := fne[0 : len(fne)-len(filepath.Ext(fne))]
	out := fmt.Sprintf("%s/%s.%s", dir, fn, config.OutputFormat)

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
func ProcessBatch(inputFiles []string, dir string, config FfmpegConfig) (Tracks, error) {
	ch := make(chan *Metadata)
	errs := make(chan error, 1)
	for _, s := range inputFiles {
		go func(s, dir string, cfg FfmpegConfig) {
			m, err := Process(s, dir, cfg)
			errs <- err
			ch <- m
		}(s, dir, config)
	}

	var tracks Tracks
	for range inputFiles {
		m := <-ch
		tracks = append(tracks, m)
	}
	return tracks, <-errs
}

type Metadata struct {
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Title       string `json:"title"`
	Track       string `json:"track"`
	Duration    string `json:"duration"`
	Filename    string `json:"filename"`
	Processed   bool   `json:"processed"`
	Transcribed bool   `json:"transcribed"`
}

type Tracks []*Metadata

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
		Artist:    artist,
		Album:     album,
		Title:     title,
		Track:     track,
		Duration:  duration,
		Processed: true,
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

// UploadToGCS uploads file specified by path to Google Cloud Storage Bucket.
func UploadToGCS(path string) (string, error) {
	// TODO (Jack, 21/06/2022):
	return "", nil
}

// Transcribe runs input path (a GCS Bucket e.g. gs://...) through Google's
// Speech-To-Text API.
func Transcribe(path string) {
	// TODO (Jack, 21/06/2022):
}
