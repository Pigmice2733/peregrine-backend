#!/bin/sh
while true; do
  go install ./...
  $@ &
  PID=$!
  inotifywait --exclude "[^g][^o]$" -r -e modify .
  kill $PID
done