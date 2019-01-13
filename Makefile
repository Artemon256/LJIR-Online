all: reuploader site

reuploader:
	go build reuploader.go

site:
	go build site.go
