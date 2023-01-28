.PHONY: build
build:
	go build -o $(GOPATH)/bin/refillstruct ./cmd/refillstruct
	