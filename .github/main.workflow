workflow "Build and Test" {
  on = "push"
  resolves = ["go test"]
}

action "go build" {
  uses = "docker://golang:1.12.7"
  runs = "go"
  args = "build ./..."
}

action "go test" {
  uses = "docker://golang:1.12.7"
  runs = "go"
  args = "test -cover -race ./..."
  needs = ["go build"]
}

workflow "Docker Build and Push" {
  on = "push"
  resolves = ["GitHub Action for Docker-1"]
}

action "GitHub Action for Docker" {
  uses = "actions/docker/cli@86ff551d26008267bb89ac11198ba7f1d807b699"
  args = "build -t peregrine-backend ."
}

action "Docker Tag" {
  uses = "actions/docker/tag@86ff551d26008267bb89ac11198ba7f1d807b699"
  needs = ["GitHub Action for Docker"]
  args = "peregrine-backend pigmice2733/peregrine-backend"
}

action "Docker Registry" {
  uses = "actions/docker/login@86ff551d26008267bb89ac11198ba7f1d807b699"
  needs = ["Docker Tag"]
  secrets = ["DOCKER_PASSWORD", "DOCKER_USERNAME"]
}

action "GitHub Action for Docker-1" {
  uses = "actions/docker/cli@86ff551d26008267bb89ac11198ba7f1d807b699"
  needs = ["Docker Registry"]
  args = "push pigmice2733/peregrine-backend"
}
