# Peregrine

[![Docker Cloud](https://img.shields.io/docker/cloud/automated/pigmice2733/peregrine-backend.svg)](https://img.shields.io/docker/cloud/automated/pigmice2733/peregrine-backend.svg)
[![Docker Cloud Status](https://img.shields.io/docker/cloud/build/pigmice2733/peregrine-backend.svg)](https://img.shields.io/docker/cloud/build/pigmice2733/peregrine-backend.svg)
[![GoDoc](https://godoc.org/github.com/Pigmice2733/peregrine-backend?status.svg)](https://godoc.org/github.com/Pigmice2733/peregrine-backend)
[![Go Report Card](https://goreportcard.com/badge/github.com/Pigmice2733/peregrine-backend)](https://goreportcard.com/report/github.com/Pigmice2733/peregrine-backend)
[![GitHub](https://img.shields.io/github/license/Pigmice2733/peregrine-backend.svg)](https://github.com/Pigmice2733/peregrine-backend/blob/master/LICENSE.md)

Peregrine is a HTTP JSON API written in Go for scouting and analysis of FIRST Robotics competitions.

For a description of what scouting is, please view the [SCOUTING.md](SCOUTING.md).

## Setup

1. Install [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) and [Go](https://golang.org/doc/install) (>=1.11)

2. Clone the repo:

```
git clone git@github.com:Pigmice2733/peregrine-backend.git
```

4. Change directory to the repo root:

```
cd peregrine-backend
```

5. [Install PostgreSQL](https://www.postgresql.org/download/) and start the server.

6. Install the `peregrine` binary:

> **NOTE**: If you cloned the repo to somewhere in your GOPATH (e.g. with `go get`) you'll need to `export GO111MODULE=on`.

```
go generate ./... # neccessary to compile OpenAPI documentation into the binary
go install ./...
```

7. Create the peregrine database:

```
createdb -U postgres peregrine
```

8. Copy the config template:

```
cp template.json config.json
```

9. Modify `config.json` as neccesary. You will likely not need to change anything besides the TBA API key and the JWT secret if you followed the instructions here. You will need to go to the [TBA account page](https://www.thebluealliance.com/account) and get a read API key and set `apiKey` under the `tba` section to the read API key you register. Set the JWT secret to the output from `uuidgen -r`.

10. Download [golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cli) and run the database migrations:

```
migrate -database "postgres://postgres@localhost:5432/peregrine?sslmode=disable" -path "$(pwd)/migrations" up
```

11. Run the app:

```
peregrine config.json
```

## API Documentation

Peregrine's entire API is documented with OpenAPI 3.0.0 (previously known as Swagger). You can
view the documentation [here](http://petstore.swagger.io/?url=http://edge.api.peregrine.ga:8080/openapi.yaml#/),
or by running peregrine locally and going [here](http://petstore.swagger.io/?url=http://localhost:8080/openapi.yaml#/).
If you notice any inaccuracies please let us know so the documentation can be corrected.

## Testing

```
go test -v ./...
```

## Contributing

1. Create a branch with a name that briefly describes the feature (e.g. `report-endpoints`):

```
git checkout -b report-endpoints
```

2. Add your commits to the branch:

```
git add internal/foo/bar.go
git commit -m "Add the initial report endpoints"
```

3. Verify that your tests pass (see the [testing section](#testing)). If they don't then fix them and add a commit.

4. Push the branch to the remote github repo:

```
git push -u origin report-endpoints
```

4. Visit the project on Github, go to Pull Requests, and hit New Pull Request.
5. Fill out the template.
6. Assign relevant reviewers, assign yourself, add any applicable labels, assign any applicable projects, and hit Create Pull Request.
