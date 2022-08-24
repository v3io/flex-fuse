# Copyright 2018 Iguazio
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
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
