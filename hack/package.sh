#!/bin/bash

mkdir -p _artifacts/
${DOCKER_CMD:-docker} build --squash --output type=tar,dest=./_artifacts/KoboRoot.tar ./
gzip -f ./_artifacts/KoboRoot.tar
mv ./_artifacts/KoboRoot.tar.gz ./_artifacts/KoboRoot.tgz
