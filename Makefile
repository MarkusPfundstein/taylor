BIN_PATH=bin
BINARY="${BIN_PATH}/taylor"

PHONY=test

test: 
	go test ./... -v

all:
	go build -o ${BINARY} main.go

.PHONY: clean
clean:
	rm -f ${BINARY}
