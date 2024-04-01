BINARY_NAME=linkreaper

build:
	go build -o bin/$(BINARY_NAME) cmd/main.go

clean:
	go clean
	rm -f $(BINARY_NAME)
