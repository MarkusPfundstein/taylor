BIN_PATH=bin
BINARY="${BIN_PATH}/taylor"

all:
	go build -o ${BINARY} main.go

.PHONY: clean
clean:
	rm -f ${BINARY}
