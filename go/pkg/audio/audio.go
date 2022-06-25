package audio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
// ffmpeg config provided saving the processed file into outDir.
func Process(path string, outDir string, config FfmpegConfig) (*Metadata, error) {
	fne := path[strings.LastIndex(path, "/")+1:]
	fn := fne[0 : len(fne)-len(filepath.Ext(fne))]
	out := fmt.Sprintf("%s/%s.%s", outDir, fn, config.OutputFormat)

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
	m.Filename = out
	return m, nil
}

type Result struct {
	Metadata *Metadata
	Err      error
}

// ProcessBatch concurrently runs Process for a specified slice of input paths.
func ProcessBatch(inDir string, outDir string, config FfmpegConfig) chan Result {
	// Get all files in directory
	var inputFiles []string
	files, err := filepath.Glob(inDir + "*.*")
	if err != nil {
		log.Println(err)
	}

	inputFiles = append(inputFiles, files...)

	resultStream := make(chan Result, len(inputFiles))
	var wg sync.WaitGroup
	wg.Add(len(inputFiles))

	for _, s := range inputFiles {
		go func(s, outDir string, cfg FfmpegConfig) {
			defer wg.Done()
			m, err := Process(s, outDir, cfg)
			resultStream <- Result{m, err}
		}(s, outDir, config)
	}

	go func() {
		wg.Wait()
		close(resultStream)
	}()

	return resultStream
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

	ctx, cancel := context.WithTimeout(ctx, time.Second*240)
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

	log.Printf("File Uploaded to GCS: %+v\n", fne)
	return fmt.Sprintf("gs://%s/%s", bucketName, fne), nil
}

type Transcript struct {
	Line       int
	Time       time.Duration
	Text       string
	ChannelTag int32
	Alternate  int
	Confidence float32
}

// Transcribe runs input path (a GCS Bucket e.g. gs://...) through Google's
// Speech-To-Text API.
func Transcribe(gsUri string, md *Metadata, outDir string) (string, error) {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return "", err
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
		return "", err
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		return "", err
	}

	// Collect results into JSON
	var trans [][]*Transcript
	for i, result := range resp.Results {
		var t []*Transcript
		for j, alt := range result.Alternatives {
			t = append(t, &Transcript{
				Line:       i,
				Time:       result.ResultEndTime.AsDuration(),
				Text:       alt.Transcript,
				ChannelTag: result.ChannelTag,
				Alternate:  j,
				Confidence: alt.Confidence,
			})
		}
		trans = append(trans, t)
	}

	transJson, _ := json.Marshal(trans)
	if err != nil {
		return "", err

	}

	// Create all sub-directories if don't exist
	// dir := fmt.Sprintf("../data/transcripts/%s/%s", md.Artist, md.Album)
	dir := fmt.Sprintf("%s/transcripts/%s/%s", outDir, md.Artist, md.Album)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}
	// Write to File
	out := fmt.Sprintf("%s/%s.json", dir, md.Title)
	f, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	w.WriteString(string(transJson))
	w.Flush()

	return string(transJson[:]), nil
}
