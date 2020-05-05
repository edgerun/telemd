# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

all: test telemd

telemd:
	$(GOBUILD) -o bin/telemd -v cmd/telemd/main.go
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f bin/telemd

docker:
	scripts/docker-build.sh
