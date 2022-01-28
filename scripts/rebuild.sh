#!/bin/bash

echo "[INFO] Run prepare resources script." &&
./prepareResources.sh &&
echo "[INFO] Go back." &&
cd ./..
echo "[INFO] Turn up all containers." &&
docker-compose up
