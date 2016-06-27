package = github.com/gracig/goping
image = gracig/goping:test
binary = goping
releasedir = release
builddir = build

.PHONY: build

all: build 

build:
	mkdir -p $(builddir)
	cd cmd && go build -o $(builddir)/$(binary)
test: 
	go test ./...

release: 
	mkdir -p $(releasedir)
	GOOS=linux GOARCH=amd64 go build -o $(releasedir)/$(binary)-linux-amd64 $(package)
	GOOS=linux GOARCH=386 go build -o $(releasedir)/$(binary)-linux-386 $(package)
	GOOS=linux GOARCH=arm go build -o $(releasedir)/$(binary)-linux-arm $(package)
	GOOS=darwin GOARCH=amd64 go build -o $(releasedir)/$(binary)-darwin-amd64 $(package)
	GOOS=windows GOARCH=amd64 go build -o $(releasedir)/$(binary)-windows-amd64.exe $(package)

clean:
	rm -rf release

