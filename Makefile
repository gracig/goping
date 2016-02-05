package = github.com/gersongraciani/goping

.PHONY: release

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build -o release/goping-linux-amd64 $(package)
	GOOS=linux GOARCH=386 go build -o release/goping-linux-386 $(package)
	GOOS=linux GOARCH=arm go build -o release/goping-linux-arm $(package)