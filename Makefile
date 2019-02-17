PACKAGE_DIRECTORIES=$(shell find . -type d -not -path "./.*" -not -path "./bin")
VERSION_PACKAGE=lockerd/version

GIT_COMMIT=$(shell git rev-parse --short HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)

GO_LDFLAGS=-X $(VERSION_PACKAGE).GitCommit=$(GIT_COMMIT)$(GIT_DIRTY)

all:
	mkdir -p bin
	go build -v -o bin/lockerd -ldflags "$(GO_LDFLAGS)" .

cover:
	go test -coverprofile cover.out ./...
	go tool cover -html=cover.out -o cover.html

dist:
	GOARCH=amd64 GOOS=linux go build -v -o bin/lockerd-linux-amd64 -ldflags "$(GO_LDFLAGS)" .

fmt:
	go fmt $(PACKAGE_DIRECTORIES)

test:
	go test -v ./...

vet:
	go vet ./...

.PHONY: all cover dist fmt test vet
