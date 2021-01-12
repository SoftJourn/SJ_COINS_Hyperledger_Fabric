#!/bin/bash

export FABRIC_CFG_PATH=${PWD}/../config/
export CHANNEL_NAME=mychannel
export CHAINCODE_NAME=coins
export CHAINCODE_VERSION=2_0
export SEQUENCE=1 # CHAINCODE_VERSION as number minus 1

export PEER_TLS=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/peers/peer0.sjfabric.softjourn.if.ua/tls/ca.crt

# Export Hyperledger-specific env variables. NOTE: they override values from ../config/core.yaml so it is okay
export CORE_PEER_LOCALMSPID="CoinsMSP"
export CORE_PEER_TLS_ENABLED=true
export ORDERER_CA=${PWD}/../configurations/ordererOrganizations/sjfabric.softjourn.if.ua/orderers/orderer.sjfabric.softjourn.if.ua/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/peers/peer0.sjfabric.softjourn.if.ua/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/users/Admin@sjfabric.softjourn.if.ua/msp
export CORE_PEER_ADDRESS=localhost:7051

echo "[${CHAINCODE_NAME}] Build chaincode"
pushd ../chaincode/github.com/coins || exit
GO111MODULE=on go mod vendor
popd || exit

echo "[${CHAINCODE_NAME}] Remove existing chaincode .tar.gz"
rm -rf ${CHAINCODE_NAME}.tar.gz

echo "[${CHAINCODE_NAME}] Package chaincode"
../bin/peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz --path ${PWD}/../chaincode/github.com/coins --lang golang --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}

echo "[${CHAINCODE_NAME}] Install chaincode"
../bin/peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA}

echo "[${CHAINCODE_NAME}] Query chaincode"
getPackageId() {
  echo $(../bin/peer lifecycle chaincode queryinstalled | sed -n "/${CHAINCODE_NAME}_${CHAINCODE_VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}")
}

PACKAGE_ID=$(getPackageId)

while [ "$PACKAGE_ID" == "" ]
do
  echo "[${CHAINCODE_NAME}] Still querying..."
  PACKAGE_ID=$(getPackageId)
  sleep 1
done

echo "[${CHAINCODE_NAME}] Approve for org"
../bin/peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA} --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --init-required --package-id ${PACKAGE_ID} --sequence ${SEQUENCE}

echo "[${CHAINCODE_NAME}] Check commit readiness"
checkReadiness() {
  echo $(echo $(../bin/peer lifecycle chaincode checkcommitreadiness --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence ${SEQUENCE} --output json --init-required) | sed 's/ //g')
}

RESPONSE=$(checkReadiness)
NEEDLE="{\"approvals\":{\"${CORE_PEER_LOCALMSPID}\":true}}"
while [ "$RESPONSE" != "$NEEDLE" ]
do
  RESPONSE=$(checkReadiness)
  echo "[${CHAINCODE_NAME}] Still checking..."
  sleep 1
done

echo "[${CHAINCODE_NAME}] Commit chaincode"
../bin/peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA} --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence ${SEQUENCE} --init-required --peerAddresses localhost:7051 --tlsRootCertFiles ${PEER_TLS}

echo "[${CHAINCODE_NAME}] Query committed state"
getCommitted() {
  echo $(../bin/peer lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME})
}

NEEDLE="Committed chaincode definition for chaincode '${CHAINCODE_NAME}' on channel '${CHANNEL_NAME}': Version: ${CHAINCODE_VERSION}, Sequence: ${SEQUENCE}, Endorsement Plugin: escc, Validation Plugin: vscc, Approvals: [${CORE_PEER_LOCALMSPID}: true]"
RESPONSE=$(getCommitted)
while [ "$RESPONSE" != "$NEEDLE" ]
do
  RESPONSE=$(getCommitted)
  echo "[${CHAINCODE_NAME}] Still querying..."
  sleep 1
done

echo "[${CHAINCODE_NAME}] Invoke init method"
RESPONSE=$(../bin/peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA} --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --isInit -c '{"function":"initLedger","Args":["sj_coin", "SJCoin"]}' --peerAddresses localhost:7051 --tlsRootCertFiles ${PEER_TLS})

sleep 10

echo "[${CHAINCODE_NAME}] Register minter via web app"
TOKEN=$(echo $(curl -d '{"username":"sj_coin","orgName":"CoinsOrg"}' -H "Content-Type: application/json" -X POST "http://localhost:4000/enroll") | sed -E 's/.*"token":"?([^,"]*)"?.*/\1/')

echo "[${CHAINCODE_NAME}] Mint 10_000_000 SJCoins"
curl -d '{"fcn":"mint","args":[10000000]}' -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" -X POST "http://localhost:4000/invoke"

# TODO Handle error -> orderer.sjfabric.softjourn.if.ua     | 2020-06-19 14:45:45.878 UTC [orderer.common.broadcast] Handle -> WARN 053 Error reading from 192.168.160.1:60162: rpc error: code = Canceled desc = context canceled
