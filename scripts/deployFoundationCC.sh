#!/bin/bash

# Variables for internal use.
export FABRIC_CFG_PATH=${PWD}/../config/
export CHANNEL_NAME=mychannel # Configure more real life name for channel.
export CHAINCODE_NAME=foundation # Configure more real life name for chaincode.
export CHAINCODE_VERSION=2_0
export SEQUENCE=1 # CHAINCODE_VERSION as number minus 1
export CHAINCODE_PATH='../chaincode/github.com/foundation'
export PEER_TLS=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/peers/peer0.sjfabric.softjourn.if.ua/tls/ca.crt

# Export Hyperledger-specific env variables. NOTE: they override values from ../config/core.yaml so it is okay
export CORE_PEER_LOCALMSPID="CoinsMSP"
export CORE_PEER_TLS_ENABLED=true
export ORDERER_CA=${PWD}/../configurations/ordererOrganizations/sjfabric.softjourn.if.ua/orderers/orderer.sjfabric.softjourn.if.ua/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/peers/peer0.sjfabric.softjourn.if.ua/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/users/Admin@sjfabric.softjourn.if.ua/msp
export CORE_PEER_ADDRESS=localhost:7051

# Build chaincode
echo "[INFO] Build chaincode"
#pushd ${CHAINCODE_PATH} || exit
#GO111MODULE=on go mod vendor
#popd || exit

# Remove existing chaincode .tar.gz
echo "[INFO] Remove existing chaincode .tar.gz"
#rm -rf ${CHAINCODE_NAME}.tar.gz

# Package chaincode
echo "[INFO] Package chaincode"
#../bin/peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz --path ${PWD}/${CHAINCODE_PATH} --lang golang --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}

# Install chaincode
echo "[INFO] Install chaincode"
#../bin/peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA}

#sleep 10

# 2.
# Query chaincode
echo "[INFO] Query chaincode"
#../bin/peer lifecycle chaincode queryinstalled >&log.txt

#PACKAGE_ID=$(sed -n "/${CHAINCODE_NAME}_${CHAINCODE_VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}" log.txt)
#rm -rf log.txt

# Approve for org
echo "[INFO] Approve for org"
#../bin/peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA} --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --init-required --package-id ${PACKAGE_ID} --sequence ${SEQUENCE}

#sleep 10

# 3.
# Check commit readiness
echo "[INFO] Check commit readiness"
#../bin/peer lifecycle chaincode checkcommitreadiness --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence ${SEQUENCE} --output json --init-required

# Commit chaincode
echo "[INFO] Commit chaincode"
#../bin/peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA} --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence ${SEQUENCE} --init-required --peerAddresses localhost:7051 --tlsRootCertFiles ${PEER_TLS}

#sleep 10

# 4.
# Query committed state
echo "[INFO] Query committed state"
#../bin/peer lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME}

# Invoke init method
echo "[INFO] Invoke init method"
../bin/peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA} --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --isInit -c '{"function":"initLedger","Args":[]}' --peerAddresses localhost:7051 --tlsRootCertFiles ${PEER_TLS}
