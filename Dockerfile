FROM gersongraciani/go-alpine-vim
RUN \
go get  github.com/gersongraciani/goping ;\
go get  github.com/smartystreets/goconvey ;\
go get  golang.org/x/net || true
CMD cd /go/src/github.com/gersongraciani/goping/lab/experiment13 ; go test -v
