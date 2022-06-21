SHELL=/bin/zsh
APP_NAME=transcribe
SRC_DIR=go

.DEFAULT_GOAL=help

build: # compiles go binaries locally
	cd ${SRC_DIR} && go build -o ../bin/${APP_NAME} ./cmd/main.go

run: # run application locally
	cd bin/ && ./${APP_NAME}


help: # shows help message
	@egrep -h '\s#\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?# "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
