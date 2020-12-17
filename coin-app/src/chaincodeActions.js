'use strict';

const {Gateway, Wallets} = require('fabric-network');
const fs = require('fs');
const path = require('path');
const config = require('../config.json')

async function invokeCode(user, code, fcn, args, isObject) {
    try {
        const gateway = await loadGateway(user);
        // Get the network (channel) our contract is deployed to.
        const network = await gateway.getNetwork(config.channelName);

        // Get the contract from the network.
        const contract = network.getContract(code);
        const transaction = contract.createTransaction(fcn);
        let result;

        if (isObject) {
            result = await transaction.submit(args);
        } else {
            result = await transaction.submit(...args);
        }

        const res = result.toString();

        console.log(`Transaction has been submitted, result is: ${res}`);

        // Disconnect from the gateway.
        await gateway.disconnect();

        return {
            success: true,
            transactionID: transaction.getTransactionId(),
            payload: res ? JSON.parse(res) : res
        };

    } catch (error) {
        console.error(`Failed to submit transaction: ${error}`);
        return {
            success: false,
            message: `Error: ${error}`
        };
    }
}

async function queryCode(user, code, fcn, args, isObject) {
    try {
        const gateway = await loadGateway(user);
        // Get the network (channel) our contract is deployed to.
        const network = await gateway.getNetwork(config.channelName);

        // Get the contract from the network.
        const contract = network.getContract(code);
        const transaction = contract.createTransaction(fcn);
        let result;

        if (isObject) {
            result = await transaction.evaluate(args);
        } else {
            result = await transaction.evaluate(...args);
        }

        const res = result.toString();

        console.log(`Transaction has been evaluated, result is: ${res}`);

        // Disconnect from the gateway.
        await gateway.disconnect();

        return {
            success: true,
            transactionID: transaction.getTransactionId(),
            payload: res ? JSON.parse(res) : res
        };

    } catch (error) {
        return {
            success: false,
            message: `Error: ${error}`
        };
    }
}

async function invoke(user, fcn, args, isObject) {
    let chaincodeName = fcn == 'createFoundation'
        ? 'foundation'
        : config.chaincodeName;
    return await invokeCode(user, chaincodeName, fcn, args, isObject)
}

async function query(user, fcn, args, isObject) {
    let chaincodeName = fcn == 'getFoundations'
          ? 'foundation'
          : config.chaincodeName;
    return await queryCode(user, chaincodeName, fcn, args, isObject);
}

async function loadGateway(user) {
    // Load the network configuration
    const ccpPath = path.resolve(process.cwd(), 'connection-profile.json');
    const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

    // Create a new file system based wallet for managing identities.
    const wallet = await Wallets.newFileSystemWallet(config.keyValueStore);

    // Check to see if we've already enrolled the user.
    const identity = await wallet.get(user);
    if (!identity) {
        console.log(`An identity for the user "${user}" does not exist in the wallet`);
        throw new Error(`An identity for the user "${user}" does not exist in the wallet`)
    }

    // Create a new gateway for connecting to our peer node.
    const gateway = new Gateway();
    await gateway.connect(ccp, {wallet, identity: user, discovery: {enabled: true, asLocalhost: false}});

    return gateway;
}

exports.invoke = invoke;
exports.invokeCode = invokeCode;
exports.query = query;
exports.queryCode = queryCode;
