.PHONY: demo

build: bin
	go build -o bin/turn github.com/factorysh/turn-tool

linux:
	GOOS=linux make build

bin:
	mkdir -p bin

demo:
	turnserver -c demo/turnserver.conf
