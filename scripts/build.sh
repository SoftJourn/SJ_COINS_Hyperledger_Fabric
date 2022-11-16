#!/bin/bash

echo "[INFO] Go to gateway." &&
pushd ../gateway &&
echo "[INFO] Build gateway." &&
./gradlew build &&
popd &&
echo "[INFO] Run prepare resources script." &&
./prepareResources.sh &&
echo "[INFO] Go to root." &&
pushd .. &&
echo "[INFO] Turn up all containers." &&
docker-compose up
