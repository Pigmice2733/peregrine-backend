# Peregrine

Peregrine is a REST API server written in Go for scouting and analysis of FIRST Robotics competitions.

# Setup

### GOPATH

> ## NOTE
> These instructions assume that you already have Go installed and your environment setup.
>
> If you do not, you can get more info at [golang.org's install guide](http://golang.org/doc/install).

Pull Peregrine from github:

	go get github.com/Pigmice2733/scouting-backend

### Vendoring

Download and install dep:

	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

Install vendored dependencies:

	dep ensure

# Database Setup

# Config File

Copy `./etc/config.yaml.template` to `./etc/config.yaml` as a starting point.

# Development

All new development should be done in a branch named `<initials>-<description>`

	git checkout -b <initials>-<description>

When the feature is complete, tests pass, and you are ready for it to be merged, create a PR.

# Pull Requests

All pull requests should include:

* ` # Goal` - description of what the PR is trying to accomplish.
* ` # Testing` - description of how the PR should be tested.