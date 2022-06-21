package server_test

import (
	"bytes"
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

	"github.com/1602077/es-lyrics-db/pkg/server"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func TestProcess(t *testing.T) {
	path, _ := os.Getwd()
	fmt.Println(path)

	url := "localhost:8080/process"
	method := "POST"
	fn := "../data/01 Smoke Signals.mp3"

	req, err := CreateRequest(method, url, fn)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.Process)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("incorrect status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"artist":"Phoebe Bridgers","album":"Stranger In The Alps","title":"Smoke Signals","track":"1","duration":"324.832656","filename":"../data/uploads/01 Smoke Signals.mp3"}`

	if rr.Body.String() != expected {
		t.Errorf("incorrect body: got %v want %v", rr.Body.String(), expected)
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
