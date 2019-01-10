all: ljir reuploader
	chmod 755 run.sh

ljir:
	go build -v -x -work ljir.go

reuploader:
	go build -v -x -work reuploader.go
