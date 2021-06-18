FROM golang:1.16 as builder

ENV PROJECT_PATH=/flex-fuse

COPY cmd ${PROJECT_PATH}/cmd
COPY pkg ${PROJECT_PATH}/pkg
COPY go.mod ${PROJECT_PATH}/go.mod
COPY go.sum ${PROJECT_PATH}/go.sum

WORKDIR ${PROJECT_PATH}
RUN go build -o /fuse ${PROJECT_PATH}/cmd/fuse/main.go

FROM alpine:3.6

COPY hack/scripts/deploy.sh /usr/local/bin
COPY hack/scripts/install.sh /install.sh
COPY hack/libs /libs
COPY --from=builder /fuse /fuse

CMD ["/bin/ash","/usr/local/bin/deploy.sh"]
