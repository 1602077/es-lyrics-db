package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/1602077/es-lyrics-db/pkg/audio"
)

func TestTranscribe(t *testing.T) {
	setup()
	defer cleanup()

	url := "localhost:8080/process"
	method := "POST"
	fn := "../testdata/Smoke Signals.wav"

	req, err := CreateRequest(method, url, fn)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(Transcribe)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Logf("incorrect status code: got %v want %v", status, http.StatusOK)
		t.FailNow()
	}

	exp := audio.Metadata{
		Artist:      "Phoebe Bridgers",
		Album:       "Stranger In The Alps",
		Title:       "Smoke Signals",
		Track:       "1",
		Duration:    "324.832656",
		Filename:    "../data/uploads/01 Smoke Signals.mp3",
		Processed:   true,
		Transcribed: false,
	}

	transPath := fmt.Sprintf("../testdata/transcripts/%s/%s/%s.json", exp.Artist, exp.Album, exp.Title)
	if _, err := os.Stat(transPath); errors.Is(err, os.ErrNotExist) {
		t.Log(err)
		t.FailNow()
	}
}

// CreateRequest generates a multipart/form-data request to upload a file via curl.
func CreateRequest(method, url, filename string) (*http.Request, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	part1, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part1, file)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "multipart/form-data")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
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
