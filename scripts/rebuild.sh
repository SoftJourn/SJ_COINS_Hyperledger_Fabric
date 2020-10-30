#!/bin/bash

print "[INFO] Go to project root." &&
cd /Users/vlad/Job/SJ_COINS_Hyperledger_Fabric &&
print "[INFO] Turn down all containers." &&
docker-compose down &&
print "[INFO] Remove dev-peer images." &&
docker rmi $(docker images 'dev-peer*' -q) &&
print "[INFO] Go to containers' runtime data folder." &&
pushd /Users/vlad/Job/.tmp &&
print "[INFO] Remove all containers' runtime data." &&
rm -rvf ca peer0 orderer node_client &&
print "[INFO] Go back." &&
popd &&
print "[INFO] Remove configurations folder." &&
rm -rv configurations &&
print "[INFO] Go to scripts folder." &&
pushd $(pwd)/scripts &&
print "[INFO] Run prepare resources script." &&
./prepareResources.sh &&
print "[INFO] Go back." &&
popd &&
print "[INFO] Turn up all containers." &&
docker-compose up
