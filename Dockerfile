FROM golang:1.10.3 as builder

ENV GOPATH=/go
ENV PROJECT_PATH=${GOPATH}/src/github.com/v3io/flex-fuse

COPY cmd ${PROJECT_PATH}/cmd
COPY pkg ${PROJECT_PATH}/pkg
COPY vendor ${PROJECT_PATH}/vendor

RUN go build -o ${GOPATH}/fuse ${PROJECT_PATH}/cmd/fuse/main.go

FROM alpine:3.6

COPY hack/scripts/deploy.sh /usr/local/bin
COPY hack/scripts/install.sh /install.sh
COPY hack/libs /libs
COPY --from=builder /go/fuse /fuse

CMD ["/bin/ash","/usr/local/bin/deploy.sh"]