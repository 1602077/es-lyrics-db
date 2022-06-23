package audio

import (
	"os"
	"path"
	"runtime"
	"testing"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func TestUploadToGCS(t *testing.T) {
	bn := "music-testing"
	fn := "../data/01 Smoke Signals.mp3"
	gcsUri, err := UploadToGCS(bn, fn)
	if err != nil {
		t.Fatal(err)
	}

	expected := "gs://music-testing/01 Smoke Signals.mp3"
	if gcsUri != expected {
		t.Fatalf("UploadToGCS(%s): got: %v, expected: %v", fn, gcsUri, expected)
	}
}

func TestTranscribe(t *testing.T) {
	gsUri := "gs://music-testing/03 - Nightgowns.wav"
	err := Transcribe(gsUri)
	if err != nil {
		t.Fatal(err)
	}
}
