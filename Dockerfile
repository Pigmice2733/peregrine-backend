FROM golang:1.13 AS build

WORKDIR /src/peregrine-backend

COPY go.mod go.sum ./
RUN go mod download

ENV CGO_ENABLED 0

COPY . .

RUN go build -o /src/peregrine-backend/peregrine ./cmd/peregrine/main.go

FROM alpine:3.9

RUN apk add ca-certificates tzdata

COPY --from=build /src/peregrine-backend/peregrine /usr/local/bin/peregrine

ENTRYPOINT [ "/usr/local/bin/peregrine" ]
CMD [ "/etc/peregrine/config.json" ]