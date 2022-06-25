package audio

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

var smokeSig = &Metadata{
	Artist:      "Phoebe Bridgers",
	Album:       "Stranger In The Alps",
	Title:       "Smoke Signals",
	Track:       "1",
	Duration:    "324.832656",
	Filename:    "../testdata/processed/Smoke Signals.wav",
	Processed:   true,
	Transcribed: false,
}
var nightGowns = &Metadata{
	Artist:      "TomMisch feat. Loyle Carner",
	Album:       "Beat Tape 2",
	Title:       "Nightgowns",
	Track:       "3/12",
	Duration:    "166.191020",
	Filename:    "../testdata/processed/Nightgowns.wav",
	Processed:   true,
	Transcribed: false,
}

func TestProbeMetadata(t *testing.T) {
	input := "../testdata/Smoke Signals.mp3"

	actual, err := ProbeMetadata(input)
	if err != nil {
		t.Fatal(err)
	}

	expected := smokeSig
	expected.Filename = ""

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Probe(%s): got: %+v, expected: %+v", input, actual, expected)
	}
}

func TestProcess(t *testing.T) {
	setup()
	defer cleanup()

	input := "../testdata/Smoke Signals.mp3"
	outDir := "../testdata/processed"
	config := FfmpegConfig{
		OutputFormat: "wav",
		SampleRate:   44100,
		NumChannels:  2,
	}

	var actual *Metadata
	actual, err := Process(input, outDir, config)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	expected := smokeSig
	expected.Filename = ""

	if !reflect.DeepEqual(actual, expected) {
		t.Logf("Process(%s): got: %+v, expected: %+v", input, actual, expected)
		t.FailNow()
	}

	if _, err := os.Stat(expected.Filename); errors.Is(err, os.ErrNotExist) {
		t.Logf("ffmpeg output not found: processed file does not exist.")
		t.FailNow()
	}
}

func TestProcessBatch(t *testing.T) {
	setup()
	defer cleanup()

	inDir := "../testdata/"
	outDir := "../testdata/processed"
	config := FfmpegConfig{
		OutputFormat: "wav",
		SampleRate:   44100,
		NumChannels:  2,
	}

	rr := ProcessBatch(inDir, outDir, config)

	// Read chan result into metadata (unblocks channel)
	for range rr {
		<-rr
	}

	for _, f := range []*Metadata{smokeSig, nightGowns} {
		if _, err := os.Stat(f.Filename); errors.Is(err, os.ErrNotExist) {
			t.Logf("Processed output not found: %s.", f.Filename)
			t.FailNow()
		}
	}

}

func TestUploadToGCS(t *testing.T) {
	bn := os.Getenv("TEST_BUCKET_NAME")
	path := "../testdata/Smoke Signals.wav"
	fn := "Smoke Signals.wav"

	defer bucketCleanup(fn)

	gcsUri, err := UploadToGCS(bn, path)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	expected := "gs://music-testing/Smoke Signals.wav"
	if gcsUri != expected {
		t.Logf("UploadToGCS(%s): got: %v, expected: %v", path, gcsUri, expected)
		t.FailNow()
	}

	// Query for object in bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	b := os.Getenv("TEST_BUCKET_NAME")
	it := client.Bucket(b).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if attrs.Name == fn {
			log.Printf("Found '%s' in bucket '%s'", fn, b)
			break
		}
		if err == iterator.Done {
			t.Logf("upload to bucket failed, could not find '%s'", fn)
			t.FailNow()
		}
		if err != nil {
			t.Logf("error list objects in '%s' bucket", b)
			t.FailNow()
		}
	}
}

func TestTranscribeB(t *testing.T) {
	fn := "Smoke Signals.wav"
	bn := os.Getenv("TEST_BUCKET_NAME")
	path := "../testdata/" + fn

	UploadToGCS(bn, path)
	defer bucketCleanup(fn)

	md, err := ProbeMetadata("../testdata/" + fn)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	gsUri := "gs://music-testing/" + fn
	_, err = Transcribe(gsUri, md, "../testdata")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	transPath := fmt.Sprintf("../testdata/transcripts/%s/%s/%s.json", md.Artist, md.Album, md.Title)
	if _, err := os.Stat(transPath); errors.Is(err, os.ErrNotExist) {
		t.Log(err)
		t.FailNow()
	}
}

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

var processDir = "../testdata/processed"
var transcriptDir = "../testdata/transcripts"
var testDirs = []string{processDir, transcriptDir}

func setup() {
	for _, d := range testDirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			os.Mkdir(d, os.ModeDir)
		}
	}
}

func cleanup() {
	for _, d := range testDirs {
		os.RemoveAll(d)
	}
}

func bucketCleanup(object string) error {
	bucket := os.Getenv("TEST_BUCKET_NAME")

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	o := client.Bucket(bucket).Object(object)

	attrs, err := o.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("object.Attrs: %v", err)
	}
	o = o.If(storage.Conditions{GenerationMatch: attrs.Generation})

	if err := o.Delete(ctx); err != nil {
		return fmt.Errorf("Object(%q).Delete: %v", object, err)
	}

	log.Printf("'%s' deleted from bucket '%s' as part of bucketCleanup", object, bucket)

	cleanup()

	return nil
}
