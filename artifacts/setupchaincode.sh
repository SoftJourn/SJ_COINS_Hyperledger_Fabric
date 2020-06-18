#!/bin/bash

## MUST BE CHANGED EVERY BUILD
export CHAINCODE_VERSION=2.0

## Chaincode info - CONFIGURE ONCE
export CHAINCODE_LANG=golang
export CHAINCODE_NAME=coins
export CHAINCODE_INIT_ARGS="{'function':'initLedger','Args':[]}"
export CHAINCODE_LABEL=${CHAINCODE_NAME}_${CHAINCODE_VERSION}
export CHAINCODE_PACKAGE_NAME=${CHAINCODE_LABEL}.tar.gz
# Source path in the cli container
export CHAINCODE_SOURCE_PATH=/opt/gopath/src/github.com/chaincode/coins

# Other info - CONFIGURE ONCE
export CHANNEL_NAME=mychannel

export ORDERER=orderer.sjfabric.softjourn.if.ua
export ORDERER_ADDRESS=${ORDERER}:7050
export ORDERER_CA_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/sjfabric.softjourn.if.ua/orderers/orderer.sjfabric.softjourn.if.ua/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem

export CORE_PEER_ADDRESS=peer0.coins.sjfabric.softjourn.if.ua:7051
export CORE_PEER_LOCALMSPID="CoinsMSP"
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/coins.sjfabric.softjourn.if.ua/peers/peer0.coins.sjfabric.softjourn.if.ua/tls/server.crt
export CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/coins.sjfabric.softjourn.if.ua/peers/peer0.coins.sjfabric.softjourn.if.ua/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/coins.sjfabric.softjourn.if.ua/users/Admin@coins.sjfabric.softjourn.if.ua/msp

peer lifecycle chaincode package $CHAINCODE_PACKAGE_NAME --path $CHAINCODE_SOURCE_PATH --lang $CHAINCODE_LANG --label $CHAINCODE_VERSION

peer lifecycle chaincode install $CHAINCODE_PACKAGE_NAME

#peer lifecycle chaincode approveformyorg --orderer $ORDERER_CA_FILE --channelID $CHANNEL_NAME --name $CHAINCODE_NAME --version $CHAINCODE_VERSION --waitForEvent --sequence $CHAINCODE_VERSION

#peer lifecycle chaincode commit --orderer $ORDERER_ADDRESS --tls --cafile $ORDERER_CA_FILE --channelID $CHANNEL_NAME --name $CHAINCODE_NAME --version $CHAINCODE_VERSION --init-required

#peer chaincode invoke --orderer $ORDERER_ADDRESS --tls --cafile $ORDERER_CA_FILE --channelID $CHANNEL_NAME --name $CHAINCODE_NAME --isInit -c $CHAINCODE_INIT_ARGS
