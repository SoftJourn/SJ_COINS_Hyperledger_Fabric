#!/bin/bash

# Delete previous resources
rm -rf ${PWD}/../configurations

# Generate crypro material
../bin/cryptogen generate --config=${PWD}/../config/crypto-config.yaml --output=${PWD}/../configurations

# Generate system genesis block
../bin/configtxgen -profile SingleOrgOrdererGenesis -configPath ${PWD}/../config -channelID system-channel -outputBlock ${PWD}/../configurations/system-genesis-block/genesis.block
