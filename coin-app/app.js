'use strict';
const config = require('./config.json');

const log4js = require('log4js');

log4js.configure({
    appenders: {
        out: {type: 'console'},
        app: {type: 'file', filename: config.logFilePath}
    },
    categories: {
        default: {appenders: ['out', 'app'], level: 'debug'},
        client: {appenders: ['out', 'app'], level: config.logLevel}
    }
});

const logger = log4js.getLogger('SampleWebApp');
const express = require('express');
const bodyParser = require('body-parser');
const http = require('http');
const app = express();
const expressJWT = require('express-jwt');
const jwt = require('jsonwebtoken');
const bearerToken = require('express-bearer-token');
const cors = require('cors');

const adminActions = require('./src/adminActions.js');
const userActions = require('./src/userActions.js');
const chaincodeActions = require('./src/chaincodeActions.js');

const host = process.env.HOST || config.host;
const port = process.env.PORT || config.port;
///////////////////////////////////////////////////////////////////////////////
//////////////////////////////// SET CONFIGURATONS ////////////////////////////
///////////////////////////////////////////////////////////////////////////////
app.use(cors());

// Support parsing of application/json type post data
app.use(bodyParser.json());

// Support parsing of application/x-www-form-urlencoded post data
app.use(bodyParser.urlencoded({
    extended: false
}));

// Set secret variable
app.set('secret', 'EcEATUqDpEcc');
app.use(expressJWT({
    secret: 'EcEATUqDpEcc'
}).unless({
    path: ['/enroll']
}));
app.use(bearerToken());

app.use(function (req, res, next) {
    logger.debug(`New request for ${req.originalUrl}`);

    if (req.originalUrl.indexOf('/enroll') >= 0) {
        return next();
    }

    const token = req.token;
    jwt.verify(token, app.get('secret'), function (err, decoded) {
        if (err) {
            res.send({
                success: false,
                message: 'Failed to authenticate token. Make sure to include the ' +
                    'token returned from /enroll call in the authorization header as a Bearer token'
            });
        } else {
            // Add the decoded user name and org name to the request object for the downstream code to use
            req.username = decoded.username;
            req.orgname = decoded.orgName;

            logger.debug(`Decoded from JWT token: username - ${decoded.username}, orgName - ${decoded.orgName}`);

            return next();
        }
    });
});

///////////////////////////////////////////////////////////////////////////////
//////////////////////////////// START SERVER /////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
const server = http.createServer(app).listen(port);

logger.info('****************** SERVER STARTED ************************');
logger.info(`***************  http://${host}:${port}  ******************`);

server.timeout = 240000;

function getErrorMessage(field) {
    return {
        success: false,
        message: `Field: ${field} is missing or invalid in the request`
    };
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////// REST ENDPOINTS START HERE ///////////////////////////
///////////////////////////////////////////////////////////////////////////////

// Register and enroll user
app.post('/enroll', async function (req, res) {
    const username = req.body.username;
    const orgName = req.body.orgName;

    logger.debug(`Endpoint : /users, Username: ${username}, OrgName: ${orgName}`);

    if (!username) {
        res.statusCode = 400;
        res.json(getErrorMessage('\'username\' or \'orgName\''));
        return;
    }

    const token = jwt.sign({
        exp: Math.floor(Date.now() / 1000) + parseInt(config.jwt_expiretime),
        username: username,
        orgName: orgName
    }, app.get('secret'));

    const adminEnrollResult = await adminActions.enroll();
    if (!adminEnrollResult.success) {
        res.statusCode = 400;
        res.json(adminEnrollResult)
        return;
    }

    const userEnrollResult = await userActions.enroll(username);
    if (userEnrollResult.success) {
        res.json({token: token});
    } else {
        res.statusCode = 400;
        res.json(userEnrollResult)
    }
});

// Invoke transaction on chaincode on target peers
app.post('/invoke', async function (req, res) {
    const fcn = req.body.fcn;
    const args = req.body.args;

    logger.debug(`Invoke function: ${fcn} with arguments: ${args}`);

    if (!fcn) {
        res.statusCode = 400;
        res.json(getErrorMessage('\'fcn\''));
        return;
    }

    if (!args) {
        res.statusCode = 400;
        res.json(getErrorMessage('\'args\''));
        return;
    }

    const adminEnrollResult = await adminActions.enroll();
    if (!adminEnrollResult.success) {
        res.statusCode = 400;
        res.json(adminEnrollResult)
        return;
    }

    const userEnrollResult = await userActions.enroll(req.username);
    if (!userEnrollResult.success) {
        res.statusCode = 400;
        res.json(userEnrollResult)
        return;
    }

    let message = await chaincodeActions.invoke(req.username, fcn, args);
    res.send(message);
});

// Invoke transaction on chaincode on target peers
app.post('/query', async function (req, res) {
    const fcn = req.body.fcn;
    const args = req.body.args;

    logger.debug(`Query function: ${fcn} with arguments: ${args}`);

    if (!fcn) {
        res.statusCode = 400;
        res.json(getErrorMessage('\'fcn\''));
        return;
    }

    if (!args) {
        res.statusCode = 400;
        res.json(getErrorMessage('\'args\''));
        return;
    }

    const adminEnrollResult = await adminActions.enroll();
    if (!adminEnrollResult.success) {
        res.statusCode = 400;
        res.json(adminEnrollResult)
        return;
    }

    const userEnrollResult = await userActions.enroll(req.username);
    if (!userEnrollResult.success) {
        res.statusCode = 400;
        res.json(userEnrollResult)
        return;
    }

    let message = await chaincodeActions.query(req.username, fcn, args);
    res.send(message);
});
