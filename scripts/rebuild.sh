#!/bin/bash

cd /Users/vlad/Job/SJ_COINS_Hyperledger_Fabric &&
docker-compose down &&
pushd /Users/vlad/Job/.tmp &&
rm -rvf ca peer0 orderer node_client &&
popd &&
rm -rv configurations &&
pushd $(pwd)/scripts &&
./prepareResources.sh &&
popd &&
docker-compose up
