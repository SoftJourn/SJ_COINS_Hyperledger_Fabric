/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

const {Wallets} = require('fabric-network');
const FabricCAServices = require('fabric-ca-client');
const fs = require('fs');
const path = require('path');
const config = require('../config.json')

async function enroll(user) {
    try {
        // Load the network configuration
        const ccpPath = path.resolve(process.cwd(), 'connection-profile.json');
        const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

        // Create a new CA client for interacting with the CA.
        const caURL = ccp['certificateAuthorities'][config.caName].url;
        const ca = new FabricCAServices(caURL);

        // Create a new file system based wallet for managing identities.
        const wallet = await Wallets.newFileSystemWallet(config.keyValueStore);

        // Check to see if we've already enrolled the user.
        const userIdentity = await wallet.get(user);
        if (userIdentity) {
            console.log(`An identity for the user "${user}" already exists in the wallet`);
            return {
                success: true
            };
        }

        // Check to see if we've already enrolled the admin user.
        const adminIdentity = await wallet.get('admin');
        if (!adminIdentity) {
            console.log('An identity for the admin user "admin" does not exist in the wallet');
            return {
                success: false,
                message: 'Admin identity is not enrolled'
            };
        }

        // build a user object for authenticating with the CA
        const provider = wallet.getProviderRegistry().getProvider(adminIdentity.type);
        const adminUser = await provider.getUserContext(adminIdentity, 'admin');

        // Register the user, enroll the user, and import the new identity into the wallet.
        const secret = await ca.register({
            affiliation: 'coins.department',
            enrollmentID: user,
            role: 'client'
        }, adminUser);

        const enrollment = await ca.enroll({
            enrollmentID: user,
            enrollmentSecret: secret
        });

        const x509Identity = {
            credentials: {
                certificate: enrollment.certificate,
                privateKey: enrollment.key.toBytes(),
            },
            mspId: config.mspId,
            type: 'X.509',
        };

        await wallet.put(user, x509Identity);

        console.log(`Successfully registered and enrolled user "${user}" and imported it into the wallet`);

        return {
            success: true
        }

    } catch (error) {
        console.error(`Failed to register user "${user}": ${error}`);
        return {
            success: false,
            message: `Error: ${error}`
        }
    }
}

exports.enroll = enroll;
