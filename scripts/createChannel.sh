#!/bin/bash

export FABRIC_CFG_PATH=${PWD}/../config/
export CHANNEL_NAME=mychannel

# Export Hyperledger-specific env variables. NOTE: they override values from ../config/core.yaml so it is okay
export CORE_PEER_LOCALMSPID="CoinsMSP"
export CORE_PEER_TLS_ENABLED=true
export ORDERER_CA=${PWD}/../configurations/ordererOrganizations/sjfabric.softjourn.if.ua/orderers/orderer.sjfabric.softjourn.if.ua/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/peers/peer0.sjfabric.softjourn.if.ua/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/../configurations/peerOrganizations/sjfabric.softjourn.if.ua/users/Admin@sjfabric.softjourn.if.ua/msp
export CORE_PEER_ADDRESS=localhost:7051

# Delete previous channel configuration
rm -rf ${PWD}/../configurations/channel-artifacts

# Create channel configuration transaction
../bin/configtxgen -profile SingleOrgChannel -configPath ${PWD}/../config -outputCreateChannelTx ${PWD}/../configurations/channel-artifacts/${CHANNEL_NAME}.tx -channelID ${CHANNEL_NAME}

# Create anchor peer update transaction
../bin/configtxgen -profile SingleOrgChannel -outputAnchorPeersUpdate ${PWD}/../configurations/channel-artifacts/CoinsOrgAnchor.tx -channelID ${CHANNEL_NAME} -asOrg CoinsOrg

# Create channel
../bin/peer channel create -o localhost:7050 -c ${CHANNEL_NAME} --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua -f ${PWD}/../configurations/channel-artifacts/${CHANNEL_NAME}.tx --outputBlock ${PWD}/../configurations/channel-artifacts/${CHANNEL_NAME}.block --tls --cafile ${ORDERER_CA}

# Join channel
../bin/peer channel join -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua -b ${PWD}/../configurations/channel-artifacts/${CHANNEL_NAME}.block --tls --cafile ${ORDERER_CA}

# Update anchor peer (optional as channel created in this org)
../bin/peer channel update -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua -c ${CHANNEL_NAME} -f ../configurations/channel-artifacts/CoinsOrgAnchor.tx --tls --cafile ${ORDERER_CA}
