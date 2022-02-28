FROM golang:1.16 as builder

ENV PROJECT_PATH=/flex-fuse

WORKDIR ${PROJECT_PATH}

# copy `go.mod` for definitions and `go.sum` to invalidate the next layer
# in case of a change in the dependencies
COPY go.mod go.sum ./

RUN go mod download

# copy source tree
COPY ./pkg ./pkg
COPY ./cmd ./cmd

RUN go build -o /fuse cmd/fuse/main.go

FROM alpine:3.6

COPY hack/scripts/deploy.sh /usr/local/bin
COPY hack/scripts/install.sh /install.sh
COPY hack/libs /libs
COPY --from=builder /fuse /fuse

CMD ["/bin/ash","/usr/local/bin/deploy.sh"]
