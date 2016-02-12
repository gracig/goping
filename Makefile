package = github.com/gersongraciani/goping
image = gersongraciani/goping:test
binary = /build/release/goping

.PHONY: build

all: build 

build:
	docker build -t $(image) .
test: 
	docker run --rm  $(image)
push:
	docker push $(image)

release: 
	mkdir release
	docker run --rm -v `pwd`:/build -e "GOOS=linux" -e "GOARCH=amd64" $(image) go build -o $(binary)-linux-amd64 $(package)
	docker run --rm -v `pwd`:/build -e "GOOS=linux" -e "GOARCH=386" $(image) go build -o $(binary)-linux-386 $(package)
	docker run --rm -v `pwd`:/build -e "GOOS=linux" -e "GOARCH=arm" $(image) go build -o $(binary)-linux-arm $(package)
	docker run --rm -v `pwd`:/build -e "GOOS=darwin" -e "GOARCH=amd64" $(image) go build -o $(binary)-darwin-amd64 $(package)

clean:
	rm -rf release

