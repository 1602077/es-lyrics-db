## es-lyrics

Go-based ingestion engine for processing songs using GCP's Speech-To-Text API, results are then written to an Elastic-Search database (to be implemented).

### Setup

1. `git clone https://github.com/1602077/es-lyrics-db`
2. Create a GCP project, inside this project create a service account with access to the Text-To-Speech API and create an authorisation key for this account.
3. Move the key to the root directory of this project.
4. Create a bucket inside of Google Cloud Storage of the project.
5. Rename `.env.example` to `.env` and update the keys to your project config.
6. Start up the server using `make build && make run`.

### HTTP Requests

To POST transcription requests to the server use the following HTTP request:
```
POST /process HTTP/1.1
Host: localhost:8080
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Length: 186

----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="file"; filename="01 Smoke Signals.mp3"
Content-Type: audio/mpeg

(data)
----WebKitFormBoundary7MA4YWxkTrZu0gW
```

Or if you prefer to use cURL:
```
curl --location --request POST 'localhost:8080/process' \
    --header 'Content-Type: multipart/form-data' \
    --form 'file=@"/abs/path/to/file"'
```
