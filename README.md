# Peregrine

![CircleCI](https://circleci.com/gh/Pigmice2733/peregrine-backend.svg?style=shield&circle-token=:circle-token)
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

6. Install the `migrate` and `peregrine` binaries:

> **NOTE**: If you cloned the repo to somewhere in your GOPATH (e.g. with `go get`) you'll need to `export GO111MODULE=on`.

```
go install ./...
```

7. Create the postgres database:

```
createdb -U postgres peregrine
```

8. Copy the config template:

```
cp etc/config.yaml.template etc/config.development.yaml
```

9. Modify `etc/config.development.yaml` as neccesary. You will likely not need to change anything besides the TBA API key if you followed the instructions here. You will need to go to the [TBA account page](https://www.thebluealliance.com/account) and get a read API key and set `apiKey` under the `tba` section to the read API key you register.
10. Run the database migrations:

```
migrate -up
```

10. Run the app:

```
peregrine
```

## Testing

Peregrine has both unit tests and integration tests. Both should be passing for a new feature.

### Unit tests

Run go test:

```
go test ./...
```

### Integration (jest) tests

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
