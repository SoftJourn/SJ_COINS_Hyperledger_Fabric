#!/bin/bash

echo "[INFO] Go to project root." &&
pushd .. &&
echo "[INFO] Turn down all containers." &&
docker-compose down &&
echo "[INFO] Go to containers' runtime data folder." &&
pushd runtime &&
echo "[INFO] Remove all containers' runtime data." &&
rm -rvf ca peer0 orderer node_client &&
echo "[INFO] Go back." &&
popd &&
echo "[INFO] Remove configurations folder." &&
rm -rvf configurations &&
echo "[INFO] Remove dev-peer images." &&
for image in $(docker images 'dev-peer*' -q)
do
  docker rmi $image;
done &&
echo "[INFO] Remove gateway image." &&
for image in $(docker images 'gateway' -q)
do
  echo $image;
  docker rmi $image;
done;
