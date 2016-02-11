FROM gersongraciani/go-alpine-vim
RUN \
go get  github.com/gersongraciani/goping ;\
go get -u github.com/golang/protobuf/protoc-gen-go ;\
go get  golang.org/x/net || true ;\
mkdir /build
VOLUME ["/build" ]
CMD cd /go/src/github.com/gersongraciani/goping ; go test -v -cover -coverprofile cover.out ./shared
