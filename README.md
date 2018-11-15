<h1 align="center"><img src="https://raw.githubusercontent.com/Pigmice2733/peregrine-logo/master/logo-with-text.png" alt="Peregrine"></h1>

![CircleCI](https://circleci.com/gh/Pigmice2733/peregrine-backend.svg?style=shield&circle-token=:circle-token)
[![Go Report Card](https://goreportcard.com/badge/github.com/Pigmice2733/peregrine-backend)](https://goreportcard.com/report/github.com/Pigmice2733/peregrine-backend)
[![GitHub](https://img.shields.io/github/license/Pigmice2733/peregrine-backend.svg)](https://github.com/Pigmice2733/peregrine-backend/blob/master/LICENSE.md)

Peregrine is a REST API server written in Go for scouting and analysis of FIRST Robotics competitions.

For a description of what scouting is, please view the [SCOUTING.md](SCOUTING.md).

# Preface

Working on Peregrine requires an understanding of the command line and of git. The documentation here is written with Linux in mind (specifically Fedora), so while it is possible to work on peregrine on Windows, it's not recommended for beginners.

# Initial Setup

1. [Install Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).
2. [Install Go](https://golang.org/doc/install).
3. Clone the repo using go get:

```
go get github.com/Pigmice2733/peregrine-backend
```

4. Change directory to the repo root:

```
cd $HOME/go/src/github.com/Pigmice2733/peregrine-backend
```

# Setup

There are two ways to get Peregrine runnning for development. You can either run it in docker (recommended for beginners), or you can run it natively on your machine. If you chose the docker route, stay in this section. Otherwise, go to the [native setup section](#native-setup-not-recommended-for-beginners).

1. [Install Docker](https://docs.docker.com/install/) and start the daemon. Also, install [docker-compose](https://docs.docker.com/compose/install/).

2. Go to the [TBA account page](https://www.thebluealliance.com/account) and get a read API key. Set the TBA API key environment variable (you will need to do this each time you close and reopen your terminal):

```
export PRGN_TBA_API_KEY="your-api-key-goes-here"
```

3. Build the docker image:

```
docker-compose build
```

4. Use docker-compose to start the app:

```
docker-compose up
```

5. Party! The application will start running on port 8080. You should be able to access the API at http://localhost:8080/. The app will also expose the PostgreSQL database on port 5432.

# Native Setup (not recommended for beginners)

1. [Install PostgreSQL](https://www.postgresql.org/download/) and start the server.
2. [Install Dep](https://github.com/golang/dep#installation).
3. Run the following command to fetch vendored dependencies:

```
dep ensure
```

4. Install the `migrate` and `peregrine` binaries:

```
go install ./...
```

5. Create the postgres database:

```
sudo -iu postgres psql -c "CREATE DATABASE peregrine"
```

5. Copy the config template:

```
cp etc/config.yaml.template etc/config.development.yaml
```

6. Modify the config file as neccesary. You will need to go to the [TBA account page](https://www.thebluealliance.com/account) and get a read API key and set `apiKey` under the `tba` section to the read API key you register.
7. Run the database migrations:

```
migrate -up
```

8. Run the app:

```
peregrine
```

# Testing

Peregrine has both unit tests and integration tests. Both should be passing for a new feature.

## Unit tests

Run go test:

```
go test ./...
```

## Integration (jest) tests

You must have a server running for integration tests. See the [setup section](#setup).

1. Go to the api-tests folder:

```
cd api-tests
```

2. Install dependencies:

```
npm i
```

3. Set the `GO_ENV` environment variable if you used Docker to setup the app:

```
export GO_ENV="docker"
```

4. Run the tests:

```
npm test
```

# Contributing

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
