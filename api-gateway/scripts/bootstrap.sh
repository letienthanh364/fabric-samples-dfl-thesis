#!/usr/bin/env bash
set -euo pipefail

CHANNEL_NAME=${CHANNEL_NAME:-nebulachannel}
CC_NAME=${CHAINCODE_NAME:-gateway}
CC_VERSION=${CHAINCODE_VERSION:-1.0}
CC_SEQUENCE=${CHAINCODE_SEQUENCE:-1}
CC_SRC_PATH=${CHAINCODE_SRC_PATH:-/chaincode/asset-transfer-basic}
CC_RUNTIME_LANGUAGE=${CHAINCODE_RUNTIME_LANGUAGE:-golang}
CC_LABEL="${CC_NAME}_${CC_VERSION}"
CC_PACKAGE_PATH=/chaincode/${CC_LABEL}.tar.gz
ORDERER_CA=/organizations/ordererOrganizations/nebula.com/orderers/orderer.nebula.com/msp/tlscacerts/tlsca.nebula.com-cert.pem
GENESIS_CHANNEL_TX=/channel-artifacts/nebula-channel.tx
CHANNEL_BLOCK=/channel-artifacts/${CHANNEL_NAME}.block
READY_MARKER=${READY_MARKER:-/scripts/.bootstrap-ready}
CC_HASH_FILE=${CHAINCODE_HASH_FILE:-/chaincode/.${CC_NAME}_hash}
FORCE_CHAINCODE_REDEPLOY=${CHAINCODE_FORCE_REDEPLOY:-0}
CHAINCODE_HASH=""
COMMITTED_SEQUENCE=0

log() {
  echo "[bootstrap] $1"
}

calculateChaincodeHash() {
  if [ ! -d "${CC_SRC_PATH}" ]; then
    echo ""
    return
  fi
  tar --mtime='UTC 1970-01-01' --owner=0 --group=0 --numeric-owner --sort=name \
    -cf - -C "${CC_SRC_PATH}" . | sha256sum | awk '{print $1}'
}

recordChaincodeHash() {
  if [ -z "${CHAINCODE_HASH}" ]; then
    return
  fi
  echo "${CHAINCODE_HASH}" > "${CC_HASH_FILE}"
}

detectCommittedSequence() {
  setGlobals 0
  if peer lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CC_NAME} > /tmp/committed_${CC_NAME}.txt 2>/tmp/committed_${CC_NAME}.err; then
    local seq
    seq=$(awk '/Sequence:/ {gsub(",", "", $2); print $2; exit}' /tmp/committed_${CC_NAME}.txt)
    if [ -n "${seq}" ]; then
      COMMITTED_SEQUENCE=${seq}
    else
      COMMITTED_SEQUENCE=0
    fi
  else
    COMMITTED_SEQUENCE=0
  fi
}

detectChaincodeChange() {
  CHAINCODE_HASH=$(calculateChaincodeHash)
  if [ "${FORCE_CHAINCODE_REDEPLOY}" = "1" ]; then
    return
  fi
  if [ -z "${CHAINCODE_HASH}" ]; then
    return
  fi
  if [ -f "${CC_HASH_FILE}" ]; then
    local recorded
    recorded=$(cat "${CC_HASH_FILE}")
    if [ "${recorded}" = "${CHAINCODE_HASH}" ]; then
      return
    fi
  fi
  FORCE_CHAINCODE_REDEPLOY=1
  log "chaincode source change detected; redeployment will be forced"
}

prepareChaincodeDeployment() {
  detectCommittedSequence
  detectChaincodeChange
  if [ "${FORCE_CHAINCODE_REDEPLOY}" = "1" ] && [ "${COMMITTED_SEQUENCE}" -gt 0 ] && [ "${CC_SEQUENCE}" -le "${COMMITTED_SEQUENCE}" ]; then
    local previous=${CC_SEQUENCE}
    CC_SEQUENCE=$((COMMITTED_SEQUENCE + 1))
    log "auto-incrementing chaincode sequence from ${previous} to ${CC_SEQUENCE}"
  fi
}

setGlobals() {
  local PEER_INDEX=$1
  local PEER_ADDRESS="peer${PEER_INDEX}.org1.nebula.com:$((7051 + PEER_INDEX*1000))"
  export CORE_PEER_LOCALMSPID=Org1MSP
  export CORE_PEER_TLS_ENABLED=true
  export CORE_PEER_MSPCONFIGPATH=/organizations/peerOrganizations/org1.nebula.com/users/Admin@org1.nebula.com/msp
  export CORE_PEER_TLS_ROOTCERT_FILE=/organizations/peerOrganizations/org1.nebula.com/peers/peer${PEER_INDEX}.org1.nebula.com/tls/ca.crt
  export CORE_PEER_ADDRESS=${PEER_ADDRESS}
}

waitForPeer() {
  local ADDRESS=$1
  local HOST=${ADDRESS%:*}
  local PORT=${ADDRESS##*:}
  for i in {1..20}; do
    if (echo >/dev/tcp/${HOST}/${PORT}) >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "failed to reach ${ADDRESS}" >&2
  exit 1
}

createChannel() {
  setGlobals 0
  if peer channel getinfo -c ${CHANNEL_NAME} >/dev/null 2>&1; then
    log "channel ${CHANNEL_NAME} already exists"
    return
  fi

  log "creating channel ${CHANNEL_NAME}"
  peer channel create \
    -o orderer.nebula.com:7050 \
    --ordererTLSHostnameOverride orderer.nebula.com \
    -c ${CHANNEL_NAME} \
    -f ${GENESIS_CHANNEL_TX} \
    --outputBlock ${CHANNEL_BLOCK} \
    --tls --cafile ${ORDERER_CA}
}

joinChannel() {
  if [ ! -f ${CHANNEL_BLOCK} ]; then
    log "fetching channel block"
    setGlobals 0
    peer channel fetch 0 ${CHANNEL_BLOCK} -o orderer.nebula.com:7050 --ordererTLSHostnameOverride orderer.nebula.com -c ${CHANNEL_NAME} --tls --cafile ${ORDERER_CA}
  fi

  for idx in 0 1 2; do
    setGlobals ${idx}
    if peer channel list >/tmp/channels_${idx}.txt 2>/tmp/channels_${idx}.err && grep -q ${CHANNEL_NAME} /tmp/channels_${idx}.txt; then
      log "peer${idx} already in channel"
      continue
    fi
    log "peer${idx} joining channel"
    peer channel join -b ${CHANNEL_BLOCK}
  done
}

packageChaincode() {
  setGlobals 0
  if [ "${FORCE_CHAINCODE_REDEPLOY}" != "1" ] && [ -f ${CC_PACKAGE_PATH} ]; then
    return
  fi
  log "packaging chaincode (${CC_LABEL})"
  rm -f ${CC_PACKAGE_PATH}
  peer lifecycle chaincode package ${CC_PACKAGE_PATH} \
    --path ${CC_SRC_PATH} \
    --lang ${CC_RUNTIME_LANGUAGE} \
    --label ${CC_LABEL}
}

installChaincode() {
  for idx in 0 1 2; do
    setGlobals ${idx}
    if [ "${FORCE_CHAINCODE_REDEPLOY}" != "1" ] && peer lifecycle chaincode queryinstalled | grep -q ${CC_LABEL}; then
      log "chaincode already installed on peer${idx}"
      continue
    fi
    if [ "${FORCE_CHAINCODE_REDEPLOY}" = "1" ]; then
      log "reinstalling chaincode on peer${idx}"
    else
      log "installing chaincode on peer${idx}"
    fi
    peer lifecycle chaincode install ${CC_PACKAGE_PATH}
  done
}

getPackageID() {
  setGlobals 0
  peer lifecycle chaincode queryinstalled > /tmp/installed_chaincodes.txt
  PACKAGE_ID=$(grep ${CC_LABEL} /tmp/installed_chaincodes.txt | tail -n 1 | awk -F ',' '{print $1}' | awk '{print $3}')
  export PACKAGE_ID
}

approveChaincode() {
  setGlobals 0
  if [ "${FORCE_CHAINCODE_REDEPLOY}" != "1" ] && peer lifecycle chaincode checkcommitreadiness --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --sequence ${CC_SEQUENCE} --output json | grep -q '"Org1MSP": true'; then
    log "chaincode already approved"
    return
  fi

  log "approving chaincode"
  peer lifecycle chaincode approveformyorg \
    -o orderer.nebula.com:7050 \
    --ordererTLSHostnameOverride orderer.nebula.com \
    --channelID ${CHANNEL_NAME} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --package-id ${PACKAGE_ID} \
    --sequence ${CC_SEQUENCE} \
    --tls --cafile ${ORDERER_CA}
}

commitChaincode() {
  if [ "${FORCE_CHAINCODE_REDEPLOY}" != "1" ] && peer lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CC_NAME} | grep -q "Sequence: ${CC_SEQUENCE}"; then
    log "chaincode already committed"
    return
  fi
  log "committing chaincode"
  peer lifecycle chaincode commit \
    -o orderer.nebula.com:7050 \
    --ordererTLSHostnameOverride orderer.nebula.com \
    --channelID ${CHANNEL_NAME} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --sequence ${CC_SEQUENCE} \
    --tls --cafile ${ORDERER_CA} \
    --peerAddresses peer0.org1.nebula.com:7051 --tlsRootCertFiles /organizations/peerOrganizations/org1.nebula.com/peers/peer0.org1.nebula.com/tls/ca.crt \
    --peerAddresses peer1.org1.nebula.com:8051 --tlsRootCertFiles /organizations/peerOrganizations/org1.nebula.com/peers/peer1.org1.nebula.com/tls/ca.crt \
    --peerAddresses peer2.org1.nebula.com:9051 --tlsRootCertFiles /organizations/peerOrganizations/org1.nebula.com/peers/peer2.org1.nebula.com/tls/ca.crt
  recordChaincodeHash
}

initializeLedger() {
  setGlobals 0
  log "invoking InitLedger"
  peer chaincode invoke \
    -o orderer.nebula.com:7050 \
    --ordererTLSHostnameOverride orderer.nebula.com \
    -C ${CHANNEL_NAME} \
    -n ${CC_NAME} \
    --tls --cafile ${ORDERER_CA} \
    --peerAddresses peer0.org1.nebula.com:7051 --tlsRootCertFiles /organizations/peerOrganizations/org1.nebula.com/peers/peer0.org1.nebula.com/tls/ca.crt \
    -c '{"function":"InitLedger","Args":[]}' || true
}

main() {
  waitForPeer "peer0.org1.nebula.com:7051" || true
  waitForPeer "peer1.org1.nebula.com:8051" || true
  waitForPeer "peer2.org1.nebula.com:9051" || true
  waitForPeer "orderer.nebula.com:7050" || true
  createChannel
  joinChannel
  prepareChaincodeDeployment
  packageChaincode
  installChaincode
  getPackageID
  approveChaincode
  commitChaincode
  initializeLedger
  log "network bootstrap completed"
  touch ${READY_MARKER}
  tail -f /dev/null
}

main
