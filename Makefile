OUT_DIR := bin
OUT_BIN := $(OUT_DIR)/main

build: main.go
	go build -o ./$(OUT_BIN) .

run: build
	./$(OUT_BIN)

clean:
	rm -rf ./$(OUT_DIR)/*