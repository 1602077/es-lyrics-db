SHELL=/bin/zsh
APP_NAME=transcribe
SRC_DIR=go
TEST_FLAGS=-v

.DEFAULT_GOAL=help

include .env

build: # compiles go binaries locally
	cd ${SRC_DIR} && go build -o ../bin/${APP_NAME} ./cmd/main.go

run: # run application locally
	cd bin/ && ./${APP_NAME}

env: # load contents of .env file
	export $$(xargs <.env)

test: # run tests
	cd ${SRC_DIR} && go test ${TEST_FLAGS} ./...

help: # shows help message
	@egrep -h '\s#\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?# "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
