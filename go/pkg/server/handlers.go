package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/1602077/es-lyrics-db/pkg/audio"
	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
)

// UploadFile posts a file to server inside of ../data/upload.
func UploadFile(r *http.Request) error {
	// Limit uploads to 10 MB
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()

	log.Printf("Uploading file: %+v\n", handler.Filename)

	dst, err := os.Create(fmt.Sprintf("../data/uploads/%s", handler.Filename))
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return err
	}

	return nil
}

// Process uploads a file specified using the file tag in curl request through
// calling UploadFile, it then pre-processes the file preparing it for GCP's
// Text-to-Speech API and writes its metadata to the response.
func Process(w http.ResponseWriter, r *http.Request) {
	if err := UploadFile(r); err != nil {
		log.Printf("err|Process|UploadFile|%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Printf("err|Process|%s", err)
		return
	}
	defer file.Close()

	indir := "../data/uploads"
	outdir := "../data/processed"
	inputFile := fmt.Sprintf("%s/%s", indir, handler.Filename)

	ac := audio.FfmpegConfig{
		OutputFormat: "wav",
		SampleRate:   44100,
		NumChannels:  1,
		// BitRate:      160,
	}

	md, err := audio.Process(inputFile, outdir, ac)
	if err == jsonvalue.ErrNotFound {
		log.Printf("warn|Process|incomplete parsing of metadata for audio %s", handler.Filename)
	}
	if err != nil && err != jsonvalue.ErrNotFound {
		log.Printf("err|Process|audio.Process|%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	md.Filename = inputFile

	log.Printf("File Processed: %+v\n", handler.Filename)

	mdJson, err := json.Marshal(md)
	if err != nil {
		log.Printf("err|Process|json.Marshal|%s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(mdJson)
}
