#!/bin/bash

# Create default main channel
docker exec -e "CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/crypto/peer/tls/ca.crt" -e "CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/crypto/admin/msp" peer0.coins.sjfabric.softjourn.if.ua peer channel create -o orderer.sjfabric.softjourn.if.ua:7050 --tls --cafile /etc/hyperledger/crypto/orderer/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem -c mychannel -f /etc/hyperledger/configtx/channel.tx

# Join peer to main channel
docker exec -e "CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/crypto/peer/tls/ca.crt" -e "CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/crypto/admin/msp" peer0.coins.sjfabric.softjourn.if.ua peer channel join -o orderer.sjfabric.softjourn.if.ua:7050 --tls --cafile /etc/hyperledger/crypto/orderer/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem -b mychannel.block

# Set peer0 as anchor peer
docker exec -e "CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/crypto/peer/tls/ca.crt" -e "CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/crypto/admin/msp" peer0.coins.sjfabric.softjourn.if.ua peer channel update -o orderer.sjfabric.softjourn.if.ua:7050 --tls --cafile /etc/hyperledger/crypto/orderer/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem -c mychannel -f /etc/hyperledger/configtx/CoinsMSPanchors.tx
