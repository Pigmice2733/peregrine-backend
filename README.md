<h1 align="center"><img src="https://raw.githubusercontent.com/Pigmice2733/peregrine-logo/master/logo-with-text.png" alt="Peregrine"></h1>

![CircleCI](https://circleci.com/gh/Pigmice2733/peregrine-backend.svg?style=shield&circle-token=:circle-token)
[![Go Report Card](https://goreportcard.com/badge/github.com/Pigmice2733/peregrine-backend)](https://goreportcard.com/report/github.com/Pigmice2733/peregrine-backend)
[![GitHub](https://img.shields.io/github/license/Pigmice2733/peregrine-backend.svg)](https://github.com/Pigmice2733/peregrine-backend/blob/master/LICENSE.md)

Peregrine is a REST API server written in Go for scouting and analysis of FIRST Robotics competitions.

# Setup

### GOPATH

> **NOTE**: These instructions assume that you already have Go installed and your environment setup. If you do not, you can get more info at [golang.org's install guide](http://golang.org/doc/install).

Pull Peregrine from GitHub:

    go get github.com/Pigmice2733/peregrine-backend

### Vendoring

Download and install dep:

    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

Install vendored dependencies:

    dep ensure

### Building

    cd cmd/peregrine
    go build

# Database

 See the postgresql first steps guide here: https://wiki.postgresql.org/wiki/First_steps

    sudo -i -u postgres
    psql
    CREATE DATABASE peregrine;

Build and run migrate:

    cd cmd/migrate
    go build
    ./migrate -up -basePath ../..

# Config File

Copy `./etc/config.json.template` to `./etc/config.development.json` as a starting point.

You must set the field `tba.apiKey` to your TBA API key. If you don't have one, go to The Blue Alliance and signup/login. From the account overview page you should be able to request a read-only API key.
You must also configure the `database` section with the credentials and details of the database you are using. If you've just setup postgres the config file will likely work without any modifications.

# Environment Variables and Flags

The environment variable `GO_ENV` can be optionally used to choose which config file to use. If it is set to "developement", `./etc/config.development.json` will be used, if "production", then `./etc/config.production.json`, etc.

The flag `-basePath` will set the directory where `/etc/config.{environment}.json` is, and is available for both `peregrine` and `migrate`.

# Running

After building:

    cd cmd/peregrine
    ./peregrine

# Development

All new development should be done in a branch named `<initials>/<description>`

    git checkout -b <initials>/<description>

When the feature is complete, tests pass, and you are ready for it to be merged, create a PR.

Pull requests must have at least one approving review (ideally two), and the CircleCI tests must pass.

# Testing

You can run all peregrine-backend unit tests by simply running `go test ./...` in the root project directory. These will be the same tests that CircleCI runs so before you even _think_ about pushing a branch, make sure you've tested it.

You can run API blackbox integration tests by going to the api-tests folder and doing the following:

    yarn
    ./node_modules/.bin/jest
