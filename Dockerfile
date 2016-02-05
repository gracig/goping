FROM golang:alpine

RUN  apk --update add git && \
go get github.com/gersongraciani/goping && \
go get github.com/smartystreets/goconvey && \
go get golang.org/x/net && \
rm -rf /var/cache/apk/* && \
cd /go/src/gersongraciani/goping/experiment13

CMD ["go test -v"]
