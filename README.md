## SJ Coins

This repo includes:
 * Node.js web app to communicate with Hyperledger Fabric. **Deprecated**
 * Java application to communicate with Hyperledger Fabric. **Replaced Node.js application**
 * Source code for chaincode **Coins** (chaincode/java/coins).
 * Set of .sh scripts that are used to set up infrastructure.
 * Set of configuration files that are used to set up infrastructure.

### Prerequisites:

 * Install the latest Docker images and Binaries from https://hyperledger-fabric.readthedocs.io/en/latest/install.html
 * Copy Binaries from **bin** folder into current project.
 * Install JDK 11+.
 * Edit .env file, set COMPOSE_PROJECT_NAME to match folder name with no space and dashes, like 'coins-sjfabric' to 'coinssjfabric'.
 * Create directory for runtime files and edit .env file, set RUNTIME_ROOT to correct path to the created directory for runtime files, or set it in environment variable.

### Setup (semi-automated)

* Change working directory to 'scripts' - `cd scripts`.
* Run `./build.sh` to build all configs and start docker containers.
* Run `./createChannel.sh` to create new channel called "mychannel".
* Run `./deployCoinsCC.sh` to compile, deploy, install and instantiate chaincode. It creates a currency 'SJCoin' with 'sj_coin' as minter, then it mints 10_000_000 SJCoins to sj_coin account.

### Setup (manual)

* Change working directory to 'gateway' - `cd gateway`.
* Run `./gradlew build` to build and package gateway application to "war".
* Change working directory to 'scripts' - `cd ../scripts`.
* Run `./prepareResources.sh` to create all necessary configurations after removing old ones.
* Change working directory to project root - `cd ..`.
* Run `docker-compose up` to start network.
* Change working directory to 'scripts' (perhaps in new terminal session) - `cd scripts`.
* Run `./createChannel.sh` to create new channel called "mychannel".
* Run `./deployCoinsCC.sh` to compile, deploy, install and instantiate chaincode. It creates a currency 'SJCoin' with 'sj_coin' as minter, then it mints 10_000_000 SJCoins to sj_coin account.
 
### Stop network
 * Press **Ctrl + C** to stop running network containers.
 * Run `docker-compose down` to remove containers (optional).

## Restart network 
  * Run `docker-compose up` to start network.
  
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
  See **chaincode/java/coins/src/main/java/com/softjourn/coins/SmartContract.java** for more details
