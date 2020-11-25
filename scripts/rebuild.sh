#!/bin/bash

echo "[INFO] Run prepare resources script." &&
./prepareResources.sh &&
echo "[INFO] Go back." &&
popd &&
echo "[INFO] Turn up all containers." &&
docker-compose up
