package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/1602077/es-lyrics-db/pkg/audio"
)

// UploadFile posts a file to server inside of ../data/upload.
func UploadFile(w http.ResponseWriter, r *http.Request) {
	// Limit uploads to 10 MB
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Printf("err|UploadFile|%s", err)
		return
	}
	defer file.Close()

	log.Printf("Uploading file: %+v\n", handler.Filename)

	dst, err := os.Create(fmt.Sprintf("../data/uploads/%s", handler.Filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Process uploads a file specified using the file tag in curl request through
// calling UploadFile, it then pre-processes the file preparing it for GCP's
// Text-to-Speech API and writes its metadata to the response.
func Process(w http.ResponseWriter, r *http.Request) {
	UploadFile(w, r)

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Printf("err|Process|%s", err)
		return
	}
	defer file.Close()

	inputFile := fmt.Sprintf("../data/uploads/%s", handler.Filename)

	ac := audio.FfmpegConfig{
		OutputFormat: "wav",
		SampleRate:   44100,
		NumChannels:  2,
		BitRate:      160,
	}

	md, err := audio.Process(inputFile, ac)
	if err != nil {
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
