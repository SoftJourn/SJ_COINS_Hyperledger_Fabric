#!/bin/bash

# Delete previous resources
rm -rf ${PWD}/../configurations

export FABRIC_CFG_PATH=${PWD}/../config

# Generate crypro material
../bin/cryptogen generate --config=${PWD}/../config/crypto-config.yaml --output=${PWD}/../configurations

# Generate system genesis block
../bin/configtxgen -profile SingleOrgOrdererGenesis -channelID system-channel -outputBlock ${PWD}/../configurations/system-genesis-block/genesis.block
