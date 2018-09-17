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

# Database Setup

# Config File

Copy `./etc/config.json.template` to `./etc/config.development.json` as a starting point.

# Development

All new development should be done in a branch named `<initials>/<description>`

	git checkout -b <initials>/<description>

When the feature is complete, tests pass, and you are ready for it to be merged, create a PR.

Pull requests must have at least one approving review (ideally two), and the CircleCI tests must pass.

# Testing

You can run all peregrine-backend unit tests by simply running `go test ./...` in the root project directory. These will be the same tests that CircleCI runs so before you even *think* about pushing a branch, make sure you've tested it.

You can run API blackbox integration tests by going to the api-tests folder and doing the following:

    yarn
    ./node_modules/.bin/jest