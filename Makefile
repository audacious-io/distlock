all:

fmt:
	go fmt -w .

test:
	go test ./...

vet:
	go vet ./...

.PHONY: all fmt test vet
