BIN_NAME=linkreaper
BIN_DIR=bin

build: $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BIN_NAME) cmd/main.go

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	go clean
	rm -rf $(BIN_DIR)
