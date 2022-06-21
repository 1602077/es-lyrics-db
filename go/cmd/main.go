package main

import (
	"log"
	"net/http"

	"github.com/1602077/es-lyrics-db/pkg/server"
)

// func main() {

// 	ac := audio.FfmpegConfig{
// 		OutputFormat: "wav",
// 		SampleRate:   44100,
// 		NumChannels:  2,
// 		BitRate:      160,
// 	}

// 	fn := "01 Smoke Signals"
// 	in := fmt.Sprintf("../data/%v.mp3", fn)

// 	audio.ProcessBatch([]string{in}, ac)
// }

func main() {
	router := server.NewRouter()
	log.Println("Starting server on port: 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
