GOOS=linux
GOARCH=amd64
VERSION=local

deps:
	go mod download

build: deps
	env GOOS=${GOOS} GOARCH=${GOARCH} go build -o bin/homectl -v -ldflags "-X 'github.com/home-sol/multicast-proxy/cmd.Version=${VERSION}'"


version: build
	chmod +x ./bin/multicast-proxy
	./bin/multicast-proxy version


test: deps
	go test ./... -v $(TESTARGS) -timeout 2m


.PHONY: deps build version test