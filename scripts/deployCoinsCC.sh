#!/bin/bash

ROOT_DIR=${PWD}/..
CHANNEL_NAME=mychannel # TODO: Should it be changed to appropriate one?
CHAINCODE_NAME=coins
CHAINCODE_VERSION=33_0
SEQUENCE=32 # CHAINCODE_VERSION as number minus 1
MINTER_USERNAME='sj_coin'
ORGANISATION_NAME='CoinsOrg'
GATEWAY_ADDRESS='http://localhost:4000'

# Steps.
STEP_REMOVE_ARCHIVE=1
STEP_BUILD=2
STEP_PACK=4
STEP_INSTALL=8
STEP_APPROVE=16
STEP_COMMIT=32
STEP_INVOKE_INIT=64
STEP_REGISTER_MINTER=128
STEP_MINT=256
STEPS=0
STEPS=$((STEPS | STEP_REMOVE_ARCHIVE))
STEPS=$((STEPS | STEP_BUILD))
STEPS=$((STEPS | STEP_PACK))
STEPS=$((STEPS | STEP_INSTALL))
STEPS=$((STEPS | STEP_APPROVE))
STEPS=$((STEPS | STEP_COMMIT))
STEPS=$((STEPS | STEP_INVOKE_INIT))
STEPS=$((STEPS | STEP_REGISTER_MINTER))
#STEPS=$((STEPS | STEP_MINT))

PEER_TLS=${ROOT_DIR}/configurations/peerOrganizations/sjfabric.softjourn.if.ua/peers/peer0.sjfabric.softjourn.if.ua/tls/ca.crt
ORDERER_ADDRESS='localhost:7050'
ORDERER_HOSTNAME='orderer.sjfabric.softjourn.if.ua'

# Export Hyperledger-specific env variables. NOTE: they override values from ../config/core.yaml so it is okay
export FABRIC_CFG_PATH="${ROOT_DIR}/config/"
export CORE_PEER_LOCALMSPID="CoinsMSP"
export CORE_PEER_TLS_ENABLED=true
export ORDERER_CA=${ROOT_DIR}/configurations/ordererOrganizations/sjfabric.softjourn.if.ua/orderers/orderer.sjfabric.softjourn.if.ua/msp/tlscacerts/tlsca.sjfabric.softjourn.if.ua-cert.pem
export CORE_PEER_TLS_ROOTCERT_FILE=${ROOT_DIR}/configurations/peerOrganizations/sjfabric.softjourn.if.ua/peers/peer0.sjfabric.softjourn.if.ua/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${ROOT_DIR}/configurations/peerOrganizations/sjfabric.softjourn.if.ua/users/Admin@sjfabric.softjourn.if.ua/msp
export CORE_PEER_ADDRESS=localhost:7051

# Remove existing chaincode archive.
if (( (STEPS & STEP_REMOVE_ARCHIVE) == STEP_REMOVE_ARCHIVE ))
then
  echo "[${CHAINCODE_NAME}] Remove existing chaincode .tar.gz"
  rm -rf ${CHAINCODE_NAME}.tar.gz
fi

# Build chaincode sources.
if (( (STEPS & STEP_BUILD) == STEP_BUILD ))
then
  echo "[${CHAINCODE_NAME}] Build chaincode"
  pushd "${ROOT_DIR}/chaincode/java/coins" || exit
  ./gradlew clean build installDist
  popd || exit
fi


# Package chaincode.
if (( (STEPS & STEP_PACK) == STEP_PACK ))
then
  echo "[${CHAINCODE_NAME}] Package chaincode"
  "${ROOT_DIR}/bin/peer" lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz \
    --path "${ROOT_DIR}/chaincode/java/coins" \
    --lang java \
    --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}
fi

# Install packaged chaincode.
if (( (STEPS & STEP_INSTALL) == STEP_INSTALL ))
then
  echo "[${CHAINCODE_NAME}] Install chaincode"
  "${ROOT_DIR}/bin/peer" lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz \
    -o ${ORDERER_ADDRESS} \
    --ordererTLSHostnameOverride ${ORDERER_HOSTNAME} \
    --tls \
    --cafile "${ORDERER_CA}"


  # Check if chaincode is installed and get package id.
  echo "[${CHAINCODE_NAME}] Query chaincode"
  getPackageId() {
    echo $("${ROOT_DIR}/bin/peer" lifecycle chaincode queryinstalled | sed -n "/${CHAINCODE_NAME}_${CHAINCODE_VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}")
  }

  PACKAGE_ID=$(getPackageId)
  while [ "$PACKAGE_ID" == "" ]
  do
    echo "[${CHAINCODE_NAME}] Retrying querying of installed chaincode status..."
    PACKAGE_ID=$(getPackageId)
    sleep 1
  done
fi


# Approve chaincode.
if (( (STEPS & STEP_APPROVE) == STEP_APPROVE ))
then
  echo "[${CHAINCODE_NAME}] Approve for org"
  "${ROOT_DIR}/bin/peer" lifecycle chaincode approveformyorg \
    -o ${ORDERER_ADDRESS} \
    --ordererTLSHostnameOverride ${ORDERER_HOSTNAME} \
    --tls \
    --cafile "${ORDERER_CA}" \
    --channelID ${CHANNEL_NAME} \
    --name ${CHAINCODE_NAME} \
    --version ${CHAINCODE_VERSION} \
    --init-required \
    --package-id "${PACKAGE_ID}" \
    --sequence ${SEQUENCE}


  # Check approve status.
  echo "[${CHAINCODE_NAME}] Check commit readiness"
  checkReadiness() {
    echo $(echo $("${ROOT_DIR}/bin/peer" lifecycle chaincode checkcommitreadiness --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence ${SEQUENCE} --output json --init-required) | sed 's/ //g')
  }

  RESPONSE=$(checkReadiness)
  NEEDLE="{\"approvals\":{\"${CORE_PEER_LOCALMSPID}\":true}}"
  while [ "$RESPONSE" != "$NEEDLE" ]
  do
    echo "[${CHAINCODE_NAME}] Trying to query approval status one more time..."
    RESPONSE=$(checkReadiness)
    sleep 1
  done
fi


# Commit chaincode.
if (( (STEPS & STEP_COMMIT) == STEP_COMMIT ))
then
  echo "[${CHAINCODE_NAME}] Commit chaincode"
  "${ROOT_DIR}/bin/peer" lifecycle chaincode commit \
    -o ${ORDERER_ADDRESS} \
    --ordererTLSHostnameOverride ${ORDERER_HOSTNAME} \
    --tls \
    --cafile "${ORDERER_CA}" \
    --channelID ${CHANNEL_NAME} \
    --name ${CHAINCODE_NAME} \
    --version ${CHAINCODE_VERSION} \
    --sequence ${SEQUENCE} \
    --init-required \
    --peerAddresses ${CORE_PEER_ADDRESS} \
    --tlsRootCertFiles "${PEER_TLS}"


  # Check commit status.
  echo "[${CHAINCODE_NAME}] Query committed state"
  getCommitted() {
    echo $("${ROOT_DIR}/bin/peer" lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME})
  }

  NEEDLE="Committed chaincode definition for chaincode '${CHAINCODE_NAME}' on channel '${CHANNEL_NAME}': Version: ${CHAINCODE_VERSION}, Sequence: ${SEQUENCE}, Endorsement Plugin: escc, Validation Plugin: vscc, Approvals: [${CORE_PEER_LOCALMSPID}: true]"
  RESPONSE=$(getCommitted)
  while [ "$RESPONSE" != "$NEEDLE" ]
  do
    echo "[${CHAINCODE_NAME}] Trying to query commit status one more time..."
    RESPONSE=$(getCommitted)
    sleep 1
  done
fi

# Invoke init method.
if (( (STEPS & STEP_INVOKE_INIT) == STEP_INVOKE_INIT ))
then
  echo "[${CHAINCODE_NAME}] Invoke init method"
  RESPONSE=$("${ROOT_DIR}/bin/peer" chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.sjfabric.softjourn.if.ua --tls --cafile ${ORDERER_CA} --channelID ${CHANNEL_NAME} --name ${CHAINCODE_NAME} --isInit -c '{"function":"initLedger","Args":["sj_coin", "SJCoin"]}' --peerAddresses localhost:7051 --tlsRootCertFiles ${PEER_TLS})

  # Lets give some time for init method to do its job.
  echo "[${CHAINCODE_NAME}] Init method has been invoked. Waiting some time for Fabric to finish its job."
  sleep 10
fi

if (( (STEPS & STEP_REGISTER_MINTER) == STEP_REGISTER_MINTER ))
then
  echo "[${CHAINCODE_NAME}] Register minter via web app"
  TOKEN=$(echo $(curl -sS -d "{\"username\":\"${MINTER_USERNAME}\",\"orgName\":\"${ORGANISATION_NAME}\"}" -H "Content-Type: application/json" -X POST "${GATEWAY_ADDRESS}/enroll") | sed -E 's/.*"token":"?([^,"]*)"?.*/\1/')
fi

if (( (STEPS & STEP_MINT) == STEP_MINT && (STEPS & STEP_REGISTER_MINTER) == STEP_REGISTER_MINTER ))
then
  echo "[${CHAINCODE_NAME}] Mint 10_000_000 SJCoins"
  curl -d '{"fcn":"mint","args":[10000000]}' -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" -X POST "${GATEWAY_ADDRESS}/invoke"
fi

if (( (STEPS & STEP_REMOVE_ARCHIVE) == STEP_REMOVE_ARCHIVE ))
then
  echo "[${CHAINCODE_NAME}] Remove existing chaincode .tar.gz"
  rm -rf ${CHAINCODE_NAME}.tar.gz
fi

# TODO Handle error -> orderer.sjfabric.softjourn.if.ua     | 2020-06-19 14:45:45.878 UTC [orderer.common.broadcast] Handle -> WARN 053 Error reading from 192.168.160.1:60162: rpc error: code = Canceled desc = context canceled
