#!/bin/bash

echo "[INFO] Go to project root." &&
cd /Users/vlad/Job/SJ_COINS_Hyperledger_Fabric &&
echo "[INFO] Turn down all containers." &&
docker-compose down &&
echo "[INFO] Remove dev-peer images." &&
docker rmi $(docker images 'dev-peer*' -q) &&
echo "[INFO] Go to containers' runtime data folder." &&
pushd /Users/vlad/Job/.tmp &&
echo "[INFO] Remove all containers' runtime data." &&
rm -rvf ca peer0 orderer node_client &&
echo "[INFO] Go back." &&
popd &&
echo "[INFO] Remove configurations folder." &&
rm -rv configurations