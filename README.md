## SJ Coins

This repo includes:
 * Node.js web app to communicate with Hyperledger Fabric 
 * Source code for chaincode **Coins**
 * Set of .sh scripts that are used to set up infrastructure
 * Set of configuration files that are used to set up infrastructure

### Prerequisites and setup:

 * Install the latest Docker images and Binaries from https://hyperledger-fabric.readthedocs.io/en/latest/install.html
 * Copy Binaries from **bin** folder into current project
 * Run **scripts/prepareResources.sh** to create all necessary configurations
 * Edit .env file, set COMPOSE_PROJECT_NAME to match folder name with no space and dashes, like 'coins-sjfabric' to 'coinssjfabric'  
 * Run **docker-compose up** to start network
 * Run **scripts/createChannel.sh** to create mychannel
 * Run **scripts/deployCC.sh** to compile, deploy, install and instantiate chaincode. It creates a currency 'SJCoin' with 'sj_coin' as minter, then it mints 10_000_000 SJCoins to sj_coin account
 
### Stop network
 * Press **Ctrl + C** to stop running network containers
 * Run **docker-compose down** to remove containers (optional)

## Restart network 
  * Run **docker-compose up** to start network
  
## Network overview
 * One Certificate Authority node - **ca.sjfabric.softjourn.if.ua**
 * One Order node - **orderer.sjfabric.softjourn.if.ua**
 * One Peer node - **peer0.sjfabric.softjourn.if.ua**
 * One CouchDB node - **couchdb0.sjfabric.softjourn.if.ua**, access: http://localhost:5984/_utils, credentials: admin/adminpw
 * One Node.js app node - **node.sjfabric.softjourn.if.ua**, access: http://localhost:4000

## API Overview
 * POST /enroll 
 
`{
   "username": "user",
   "orgName": "CoinsOrg"
 }`
 
 * POST /invoke (isObject - true if args is an object and should be converted to JSON on chaincode invocation)

`{
    "fcn": "BatchTransfer",
    "args": [{"userId": "dude1","amount": 100}],
    "isObject": true
 }`
  * POST /query 
 
`{
    "fcn": "GetBalance",
    "args": "user"
 }`
 
 ### Chaincode overview
  See **chaincode/github.com/coins/coin.go** for more details
