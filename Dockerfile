FROM golang:alpine

WORKDIR /go/src/github.com/Pigmice2733/peregrine-backend
COPY . .

RUN apk add inotify-tools curl git bash

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure -vendor-only
RUN go install ./...