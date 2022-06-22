package audio

import "testing"

func TestTranscribe(t *testing.T) {
	gsUri := "gs://music-testing/01 - The Journey.wav"
	err := Transcribe(gsUri)
	if err != nil {
		t.Fatal(err)
	}
}
