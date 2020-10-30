#!/bin/bash

echo "[INFO] Go to project root." &&
cd /Users/vlad/Job/SJ_COINS_Hyperledger_Fabric &&
echo "[INFO] Go to scripts folder." &&
pushd $(pwd)/scripts &&
echo "[INFO] Run prepare resources script." &&
./prepareResources.sh &&
echo "[INFO] Go back." &&
popd &&
echo "[INFO] Turn up all containers." &&
docker-compose up
