FROM golang

ADD . /go/src/github.com/Pigmice2733/peregrine-backend

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/github.com/Pigmice2733/peregrine-backend

RUN dep ensure -vendor-only

RUN go install github.com/Pigmice2733/peregrine-backend/cmd/peregrine

ENTRYPOINT /go/bin/peregrine

EXPOSE 8080