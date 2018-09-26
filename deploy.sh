#!/bin/bash

pkill peregrine

rm -rf deploy
tar xf deploy.tgz

mkdir deploy/etc
cp ./config.production.yaml deploy/etc

export GO_ENV="production"
cd ./deploy
./bin/migrate -up

echo 'a'
nohup ./bin/peregrine &