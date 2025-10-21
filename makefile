# Makefile simple para Windows
build:
	go build -o bin/main.exe ./cmd/...

run: build
	cd bin && ./main.exe
