GOOS=linux
GOARCH=amd64
VERSION=local

deps:
	go mod download

build: deps
	env GOOS=${GOOS} GOARCH=${GOARCH} go build -o bin/multicast-proxy -v -ldflags "-X 'github.com/home-sol/multicast-proxy/cmd.Version=${VERSION}'"
	sudo setcap cap_net_raw,cap_net_admin=eip bin/multicast-proxy


version: build
	chmod +x ./bin/multicast-proxy
	./bin/multicast-proxy version


test: deps
	go test ./... -v $(TESTARGS) -timeout 2m

docker-build: build
	docker build -f ./Dockerfile --tag multicast-proxy:local ./bin


.PHONY: deps build version test