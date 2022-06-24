package audio

import (
	"errors"
	"fmt"
	"log"
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
	md, err := ProbeMetadata("../data/03 - Nightgowns")
	if err != nil {
		t.Fatal(err)
	}

	gsUri := "gs://music-testing/03 - Nightgowns.wav"
	_, err = Transcribe(gsUri, md)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrite(t *testing.T) {
	dir := "../data/transcript"
	out := fmt.Sprintf("%s/%s.json", dir, "test")
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
	// f, err := os.Create(out)
	f, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

}
