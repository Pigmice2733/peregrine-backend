<h1 align="center"><img src="https://raw.githubusercontent.com/Pigmice2733/peregrine-logo/master/logo-with-text.png" alt="Peregrine"></h1>

![CircleCI](https://circleci.com/gh/Pigmice2733/peregrine-backend.svg?style=shield&circle-token=:circle-token)
[![Go Report Card](https://goreportcard.com/badge/github.com/Pigmice2733/peregrine-backend)](https://goreportcard.com/report/github.com/Pigmice2733/peregrine-backend)
[![GitHub](https://img.shields.io/github/license/Pigmice2733/peregrine-backend.svg)](https://github.com/Pigmice2733/peregrine-backend/blob/master/LICENSE.md)

Peregrine is a REST API server written in Go for scouting and analysis of FIRST Robotics competitions.

# Scouting

## What's Scouting?

Scouting is when teams observe the behaviors of a team’s robot throughout matches they compete in within an event. When scouting other teams, scouts look for how a robot performs certain actions during a match, the overall functionality of the team's robot, and how efficient the performance of drive teams are.

## Why Do We Scout?

The most basic reason we scout is that data we receive from scouting assists us in alliance selections, where highly ranked teams select other teams to join their alliance and compete in playoff matches in hopes to exit the event as the finalist winner. The alliance captains, those who choose other robots to join their alliance, rely on scouting data to strategize which teams will complement their robot's abilities in a game. Cooperating well as a team is critical to playoff matches, as not doing so can cost an alliance many points in a match. Before a match, it's helpful for Drive Captains to know which teams they are competing against as well as who is on their alliance. Scouting gives Drive Captains that insight, and they use the data to strategize with other teams on their alliance to win the upcoming match.

## Why Use a Scouting App?

A scouting app solves many problems found in traditional paper scouting methods. Most FRC teams use a certain, inefficient process to scout matches; while matches are playing, scouts write data onto paper by checking boxes or filling in numbers based on how a robot competed during the match. Afterward, the data is manually entered into a spreadsheet, tending to be disorganized and difficult to interpret. Because manually entering data into a spreadsheet takes up extra time, the process of strategizing for alliance selections is delayed. In addition, a disorderly spreadsheet makes tracking a team's progress throughout an event challenging. Through a scouting app, the process of manually entering data after matches is eliminated, which allows for the pre-alliance selection process to begin much earlier. The data is immediately uploaded to an organized, easy-to-follow spreadsheet, and multiple graphs are created that display a robot's progress during an event. These components allow for Drive Captains to analyze their opponents quickly.

# Initial Setup

### GOPATH

> **NOTE**: These instructions assume that you already have Go installed and your environment setup. If you do not, you can get more info at [golang.org's install guide](http://golang.org/doc/install).

Pull Peregrine from GitHub:

    go get github.com/Pigmice2733/peregrine-backend

# Quickstart (development)

### Configuration

A TBA (the blue alliance) API key is needed to fetch events and matches from the blue alliance. You can get one by visiting [the account page](https://www.thebluealliance.com/account). Go to the "Read API Keys" section, write a short description (e.g. "Peregrine Development Instance"), and hit "Add New Key". Then, copy the key in it's entirety, and run the following command:

    export TBA_API_KEY="your-api-key-here"

### Running

> **NOTE**: These instructions assume that you already have Docker installed and have the daemon running. If you do not, you can get more info at [docker's install guide](https://docs.docker.com/install/).

> **TIP**: If you get an error similar to `got error: opening config: open etc/config.docker.yaml: permission denied`, you may need to run the following command to set selinux to permissive mode: `sudo setenforce 0`.

You can run peregrine-backend in docker for development. While it's a bit slower, it can allow you to get started in just one command. From the base application directory:

    docker-compose up

The application will rebuild and restart on changes of any \*.go file. If you want to run new migrations or clear your database, you can do Ctrl+C to kill docker-compose and restart it with the above command. By default it will run the server on port 8080.

# Without Docker (development)

### Vendoring

> **NOTE**: These instructions assume that you already have dep installed. If you
> do not, you can get more info at [dep's install guide](https://github.com/golang/dep/blob/master/docs/installation.md).

Install vendored dependencies:

    dep ensure

### Building

    cd cmd/peregrine
    go build

## Database

See the postgresql first steps guide here: https://wiki.postgresql.org/wiki/First_steps

    sudo -i -u postgres
    psql
    CREATE DATABASE peregrine;

Build and run migrate:

    cd cmd/migrate
    go build
    ./migrate -up -basePath ../..

## Config File

Copy `./etc/config.json.template` to `./etc/config.development.json` as a starting point.

You must set the field `tba.apiKey` to your TBA API key. If you don't have one, go to The Blue Alliance and signup/login. From the account overview page you should be able to request a read-only API key.
You must also configure the `database` section with the credentials and details of the database you are using. If you've just setup postgres the config file will likely work without any modifications.
If the `TBA_API_KEY` environment variable is set, it will override the one in the config file. This is mostly just used for CI, you shouldn't need to use it in development. To see the full config schema, you can run `go doc "peregrine-backend/internal/config".Config`

## Environment Variables and Flags

The environment variable `GO_ENV` can be optionally used to choose which config file to use. If it is set to "developement", `./etc/config.development.json` will be used, if "production", then `./etc/config.production.json`, etc.

The flag `-basePath` will set the directory where `/etc/config.{environment}.json` is, and is available for both `peregrine` and `migrate`.

## Running

After building:

    cd cmd/peregrine
    ./peregrine

# Contributing

All new development should be done in a branch named `<description>`

    git checkout -b <description>

When the feature is complete, tests pass, and you are ready for it to be merged, create a PR.

Pull requests must have at least one approving review (ideally two), and the CircleCI tests must pass.
