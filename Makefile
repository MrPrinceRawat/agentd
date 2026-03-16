.PHONY: build build-all clean

build:
	go build -o bin/agentctl ./cmd/agentctl

build-all: build
	GOOS=linux GOARCH=amd64 go build -o bin/agentd-linux-amd64 ./cmd/agentd
	GOOS=linux GOARCH=arm64 go build -o bin/agentd-linux-arm64 ./cmd/agentd

clean:
	rm -rf bin/
