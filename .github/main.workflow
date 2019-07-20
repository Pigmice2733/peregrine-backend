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
