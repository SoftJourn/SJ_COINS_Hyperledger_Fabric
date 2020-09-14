/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

const FabricCAServices = require('fabric-ca-client');
const {Wallets} = require('fabric-network');
const fs = require('fs');
const path = require('path');
const config = require('../config.json')

async function enroll() {
    try {
        // Load the network configuration
        const ccpPath = path.resolve(process.cwd(), 'connection-profile.json');
        const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

        // Create a new CA client for interacting with the CA.
        const caInfo = ccp['certificateAuthorities'][config.caName];
        const caTLSCACerts = fs.readFileSync(caInfo['tlsCACerts'].path[0]);
        const ca = new FabricCAServices(caInfo.url, {trustedRoots: Buffer.from(caTLSCACerts), verify: false}, caInfo.caName);

        // Create a new file system based wallet for managing identities.
        const wallet = await Wallets.newFileSystemWallet(config.keyValueStore);

        // Check to see if we've already enrolled the admin user.
        const identity = await wallet.get('admin');
        if (identity) {
            console.log('An identity for the admin user "admin" already exists in the wallet');
            return {
                success: true
            };
        }

        // Enroll the admin user, and import the new identity into the wallet.
        const enrollment = await ca.enroll({enrollmentID: 'admin', enrollmentSecret: 'adminpw'});
        const x509Identity = {
            credentials: {
                certificate: enrollment.certificate,
                privateKey: enrollment.key.toBytes(),
            },
            mspId: config.mspId,
            type: 'X.509',
        };
        await wallet.put('admin', x509Identity);

        console.log('Successfully enrolled admin user "admin" and imported it into the wallet');

        return {
            success: true
        }

    } catch (error) {
        console.error(`Failed to enroll admin user "admin": ${error}`);

        return {
            success: false,
            message: `Error: ${error}`
        }
    }
}

exports.enroll = enroll;
