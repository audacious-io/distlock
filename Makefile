all:

cover:
	go test -coverprofile cover.out ./...
	go tool cover -html=cover.out -o cover.html

fmt:
	go fmt . ./locking

test:
	go test -v ./...

vet:
	go vet ./...

.PHONY: all cover fmt test vet
