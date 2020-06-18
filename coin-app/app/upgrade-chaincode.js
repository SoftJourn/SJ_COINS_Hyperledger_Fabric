/**
 * Copyright 2017 IBM All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */
'use strict';
var path = require('path');
var fs = require('fs');
var util = require('util');
var hfc = require('fabric-client');
var Peer = require('fabric-client/lib/Peer.js');
var EventHub = require('fabric-client/lib/EventHub.js');
var config = require('../config.json');
var helper = require('./helper.js');
var logger = helper.getLogger('upgrade-chaincode');
hfc.addConfigFile(path.join(__dirname, 'network-config.json'));
var ORGS = hfc.getConfigSetting('network-config');
var tx_id = null;
var eh = null;

var upgradeChaincode = async function (channelName, chaincodeName, chaincodeVersion, functionName, args, username, org_name) {
    logger.debug('\n============ Upgrade chaincode on organization ' + org +
        ' ============\n');

    let error_message = null;

    try {
        // first setup the client for this org
        const client = await helper.getClientForOrg(org_name, username);
        logger.debug('Successfully got the fabric client for the organization "%s"', org_name);
        const channel = client.getChannel(channelName);
        if (!channel) {
            let message = util.format('Channel %s was not defined in the connection profile', channelName);
            logger.error(message);
            return message;
        }
        const tx_id = client.newTransactionID();
        const deployId = tx_id.getTransactionID();

        // send proposal to endorser
        const request = {
            chaincodeId: chaincodeName,
            chaincodeVersion: chaincodeVersion,
            fcn: functionName,
            args: args,
            txId: tx_id
        };

        if (functionName) {
            request.fcn = functionName;
        }

        let results = await channel.sendUpgradeProposal(request, 60000); //instantiate takes much longer

        // the returned object has both the endorsement results
        // and the actual proposal, the proposal will be needed
        // later when we send a transaction to the orderer
        const proposalResponses = results[0];
        const proposal = results[1];

        // look at the responses to see if they are all are good
        // response will also include signatures required to be committed
        let all_good = true;
        for (const i in proposalResponses) {
            if (proposalResponses[i] instanceof Error) {
                all_good = false;
                error_message = util.format('upgrade proposal resulted in an error :: %s', proposalResponses[i].toString());
                logger.error(error_message);

            } else if (proposalResponses[i].response && proposalResponses[i].response.status === 200) {
                logger.info('upgrade proposal was good');

            } else {
                all_good = false;
                error_message = util.format('upgrade proposal was bad for an unknown reason %j', proposalResponses[i]);
                logger.error(error_message);
            }
        }

        if (all_good) {
            logger.info(util.format(
                'Successfully sent Proposal and received ProposalResponse: Status - %s, message - "%s", metadata - "%s"',
                proposalResponses[0].response.status, proposalResponses[0].response.message,
                proposalResponses[0].response.payload));

            // wait for the channel-based event hub to tell us that the
            // instantiate transaction was committed on the peer
            const promises = [];
            const event_hubs = channel.getChannelEventHubsForOrg();
            logger.debug('found %s eventhubs for this organization %s', event_hubs.length, org_name);

            event_hubs.forEach((eh) => {
                let instantiateEventPromise = new Promise((resolve, reject) => {
                    logger.debug('upgradeEventPromise - setting up event');
                    let event_timeout = setTimeout(() => {
                        let message = 'REQUEST_TIMEOUT:' + eh.getPeerAddr();
                        logger.error(message);
                        eh.disconnect();
                    }, 60000);

                    eh.registerTxEvent(deployId, (tx, code, block_num) => {
                            logger.info('The chaincode upgrade transaction has been committed on peer %s', eh.getPeerAddr());
                            logger.info('Transaction %s has status of %s in blocl %s', tx, code, block_num);
                            clearTimeout(event_timeout);

                            if (code !== 'VALID') {
                                let message = util.format('The chaincode upgrade transaction was invalid, code:%s', code);
                                logger.error(message);
                                reject(new Error(message));
                            } else {
                                let message = 'The chaincode upgrade transaction was valid.';
                                logger.info(message);
                                resolve(message);
                            }
                        }, (err) => {
                            clearTimeout(event_timeout);
                            logger.error(err);
                            reject(err);
                        },
                        // the default for 'unregister' is true for transaction listeners
                        // so no real need to set here, however for 'disconnect'
                        // the default is false as most event hubs are long running
                        // in this use case we are using it only once
                        {unregister: true, disconnect: true}
                    );
                    eh.connect();
                });
                promises.push(instantiateEventPromise);
            });

            const orderer_request = {
                txId: tx_id, // must include the transaction id so that the outbound
                // transaction to the orderer will be signed by the admin id
                // the same as the proposal above, notice that transactionID
                // generated above was based on the admin id not the current
                // user assigned to the 'client' instance.
                proposalResponses: proposalResponses,
                proposal: proposal
            };
            const sendPromise = channel.sendTransaction(orderer_request);
            // put the send to the orderer last so that the events get registered and
            // are ready for the orderering and committing
            promises.push(sendPromise);
            const results = await Promise.all(promises);

            logger.debug(util.format('------->>> R E S P O N S E : %j', results));
            const response = results.pop(); //  orderer results are last in the results

            if (response.status === 'SUCCESS') {
                logger.info('Successfully sent transaction to the orderer.');
            } else {
                error_message = util.format('Failed to order the transaction. Error code: %s', response.status);
                logger.debug(error_message);
            }

            // now see what each of the event hubs reported
            for (const i in results) {
                const event_hub_result = results[i];
                const event_hub = event_hubs[i];
                logger.debug('Event results for event hub :%s', event_hub.getPeerAddr());
                if (typeof event_hub_result === 'string') {
                    logger.debug(event_hub_result);
                } else {
                    if (!error_message) error_message = event_hub_result.toString();
                    logger.debug(event_hub_result.toString());
                }
            }
        }
    } catch (error) {
        logger.error('Failed to send instantiate due to error: ' + error.stack ? error.stack : error);
        error_message = error.toString();
    } finally {
        if (channel) {
            channel.close();
        }
    }

    let message = util.format('Successfully upgrade chaincode in organization %s to the channel \'%s\'', channelName);
    if (error_message) {
        message = util.format('Failed to upgrade the chaincode. cause:%s', error_message);
        logger.error(message);

    } else {
        logger.info(message);
    }

    return message;
};

exports.upgradeChaincode = upgradeChaincode;
