FROM golang:alpine

RUN  apk --update add git ;\
go get  github.com/gersongraciani/goping ;\
go get  github.com/smartystreets/goconvey ;\
go get  golang.org/x/net ;\
rm -rf /var/cache/apk/* 

CMD cd /go/src/github.com/gersongraciani/goping/lab/experiment13 ; go test -v
