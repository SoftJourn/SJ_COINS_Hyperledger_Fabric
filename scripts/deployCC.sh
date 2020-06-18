#!/bin/bash

#pushd ../artifacts/src/github.com/coins || exit
#GO111MODULE=on go mod vendor
#popd || exit

export FABRIC_CFG_PATH=${PWD}/../artifacts/configtx

export CORE_PEER_LOCALMSPID="CoinsMSP"
export CORE_PEER_ADDRESS=localhost:7051

export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../artifacts/channel/crypto-config/peerOrganizations/coins.sjfabric.softjourn.if.ua/peers/peer0.coins.sjfabric.softjourn.if.ua/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/../artifacts/channel/crypto-config/peerOrganizations/coins.sjfabric.softjourn.if.ua/users/Admin@coins.sjfabric.softjourn.if.ua/msp
export CA_CERT=${PWD}/../artifacts/channel/crypto-config/ordererOrganizations/sjfabric.softjourn.if.ua/orderers/orderer.sjfabric.softjourn.if.ua/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem

# Package chaincode
#../bin/peer lifecycle chaincode package coins.tar.gz --path ${PWD}/../artifacts/src/github.com/coins --lang golang --label coins_2_0

# Install chaincode
#../bin/peer lifecycle chaincode install coins.tar.gz

## Query chaincode
#../bin/peer lifecycle chaincode queryinstalled

## Approve for org
#../bin/peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${CA_CERT} --channelID mychannel --name coins --version 2_0 --init-required --sequence 2_0

## Invoke init method
#../bin/peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${CA_CERT} --channelID mychannel --name coins --isInit -c '{"function":"initLedger","Args":[]}'
