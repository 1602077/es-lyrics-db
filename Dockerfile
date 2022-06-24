FROM golang:1.18 as builder
WORKDIR /app
COPY go/go.mod go/go.sum ./
RUN go mod download && go mod verify
COPY go/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o transcribe -a -installsuffix cgo ./cmd/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates ffmpeg
WORKDIR /app/data
RUN mkdir -p uploads proccessed transcripts
WORKDIR /app/
COPY .env.docker ./.env
COPY creds.json .
WORKDIR /app/bin
COPY --from=builder /app/transcribe .
EXPOSE 8080
CMD ["./transcribe"]
