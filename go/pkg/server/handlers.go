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
	"github.com/joho/godotenv"
)

// UploadToServer posts a file to server inside of ../data/upload.
func UploadToServer(r *http.Request) error {
	// Limit uploads to 40 MB
	r.ParseMultipartForm(40 << 20)

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

// Preprocess each file uploaded using UploadToServer using ffmpeg and
// ffmpegprobe. This prepares the file to be passed to Google's Text-to-Speech
// API and extracts metadata (artist, album, trackname) from the input file.
func Preprocess(r *http.Request) (*audio.Metadata, error) {
	if err := UploadToServer(r); err != nil {
		return &audio.Metadata{}, err
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		return &audio.Metadata{}, err
	}
	defer file.Close()

	indir := "../data/uploads"
	outdir := "../data/processed"
	inputFile := fmt.Sprintf("%s/%s", indir, handler.Filename)

	ac := audio.FfmpegConfig{
		OutputFormat: "wav",
		SampleRate:   44100,
		NumChannels:  2,
		// BitRate:      160,
	}

	md, err := audio.Process(inputFile, outdir, ac)
	if err == jsonvalue.ErrNotFound {
		log.Printf("warn|Process|incomplete parsing of metadata for audio %s", handler.Filename)
	}
	if err != nil && err != jsonvalue.ErrNotFound {
		return &audio.Metadata{}, err
	}

	md.Filename = inputFile

	log.Printf("File Processed: %+v\n", handler.Filename)

	return md, nil
}

// Transcribe calls Google's Text-To-Speech API for the provided file.
// Each file goes through the following pipeline:
// UploadToServer -> Preprocess -> UploadToGCS -> Transcribe
func Transcribe(w http.ResponseWriter, r *http.Request) {
	var md *audio.Metadata
	md, err := Preprocess(r)
	if err != nil {
		log.Printf("err|Transcribe|UploadFile|%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = godotenv.Load("../.env")
	if err != nil {
		log.Printf("err|godotenv.Load|error opening .env file")
	}

	gsUri, err := audio.UploadToGCS(os.Getenv("BUCKET_NAME"), md.Filename)
	if err != nil {
		log.Printf("err|Transcribe|a.UploadToGCS|%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	transcript, err := audio.Transcribe(gsUri, md, "../data")
	if err != nil {
		log.Printf("err|Transcribe|a.Transcribe|%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	md.Transcribed = true

	mdJson, err := json.Marshal(md)
	if err != nil {
		log.Printf("err|Transcribe|json.Marshal|%s", err)
		return
	}

	response := fmt.Sprintf("[{%v, \"transcript\": %v]", string(mdJson), transcript)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))

	log.Printf("File Transcribed: %+v\n", md.Filename)
}
