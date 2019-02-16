all:

fmt:
	go fmt .

test:
	go test ./...

vet:
	go vet ./...

.PHONY: all fmt test vet
