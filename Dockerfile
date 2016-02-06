FROM gersongraciani/go-alpine-vim
RUN \
go get  github.com/gersongraciani/goping ;\
go get  golang.org/x/net || true
CMD cd /go/src/github.com/gersongraciani/goping ; go test -v -cover -coverprofile cover.out ./shared
