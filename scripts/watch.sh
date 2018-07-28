#!/bin/bash

while true; do
  go install github.com/Pigmice2733/peregrine-backend/cmd/peregrine
  peregrine &
  PID=$!
  trap "kill $PID; exit" SIGINT
  inotifywait -r -e modify $GOPATH/src/github.com/Pigmice2733/peregrine-backend
  kill $PID
done
