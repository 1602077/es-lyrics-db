package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/storage"
	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/joho/godotenv"
	fluentffmpeg "github.com/modfy/fluent-ffmpeg"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
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
		return m, err
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
	dd, err := json.Marshal(d)
	if err != nil {
		return &Metadata{}, err
	}

	j, err := jsonvalue.Unmarshal(dd)
	if err != nil {
		return &Metadata{}, err
	}

	// Parse Metadata from un-marshalled json
	var errTargetNotFound error

	artist, err := j.GetString("format", "tags", "artist")
	if err != nil {
		errTargetNotFound = err
	}

	album, err := j.GetString("format", "tags", "album")
	if err != nil {
		errTargetNotFound = err
	}

	title, err := j.GetString("format", "tags", "title")
	if err != nil {
		errTargetNotFound = err
	}

	track, err := j.GetString("format", "tags", "track")
	if err != nil {
		errTargetNotFound = err
	}

	duration, err := j.GetString("format", "duration")
	if err != nil {
		errTargetNotFound = err
	}

	return &Metadata{
		Artist:    artist,
		Album:     album,
		Title:     title,
		Track:     track,
		Duration:  duration,
		Processed: true,
	}, errTargetNotFound
}

// UploadToGCS uploads file specified by path to Google Cloud Storage Bucket.
func UploadToGCS(bucketName, filePath string) (string, error) {
	// Load in GOOGLE_APPLICATION_CREDENTIALS defined in .env
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("err|godotenv.Load|error opening .env file")
	}

	// Create client to write to BucketName with
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("os.Open: %v", err)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	fne := filePath[strings.LastIndex(filePath, "/")+1:]

	// Write file to GCS Bucket
	wc := client.Bucket(bucketName).Object(fne).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return "", fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %v", err)
	}

	return fmt.Sprintf("gs://%s/%s", bucketName, fne), nil
}

// Transcribe runs input path (a GCS Bucket e.g. gs://...) through Google's
// Speech-To-Text API.
func Transcribe(gsUri string) error {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return err
	}

	// Generate transcription job config
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			AudioChannelCount:                   2,
			EnableSeparateRecognitionPerChannel: true,
			Encoding:                            speechpb.RecognitionConfig_LINEAR16,
			// SampleRateHertz: 44100,
			LanguageCode: "en-GB",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gsUri},
		},
	}

	// Trigger job
	op, err := client.LongRunningRecognize(ctx, req)
	if err != nil {
		return err
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		return err
	}

	// Write transcript to file
	// TODO (Jack, 23/06/2022): output responses to a file instead
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			log.Printf("\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
		}
	}
	return nil
}
