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
//////////////////////////////// SET CONFIGURATIONS ////////////////////////////
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
    secret: 'EcEATUqDpEcc',
    algorithms: ['sha1', 'RS256', 'HS256']
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

// WARN: This is quick but not good solution for sequential enroll requests.
var EnrollmentDecision = {
    queue: {},

    init: function(id) {
        if (!this.queue[id]) {
            this.queue[id] = [];
        }

        let uuid = this.getUUID();
        this.queue[id].push({uuid: uuid, time: +new Date()});

        return uuid;
    },

    acquire: function(id, uuid) {
        let queue = this.queue[id];

        while (queue.length > 0) {
            let next = queue[0];

            if (next.uuid != uuid) {
                if (next.time < ((+new Date()) - 3000)) {
                    queue.shift();
                    continue;
                } else {
                    return false;
                }
            } else {
                return true;
            }
        }

        return true;
    },

    clear: function(id, uuid) {
        if (!this.queue[id]) {
            return;
        }

        let index = this.queue[id].findIndex(function (e) { return e.uuid == uuid; });
        if (index > -1) {
            this.queue[id].splice(index, 1);
        }
    },

    getUUID: function() {
      return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
      });
    }
};

///////////////////////////////////////////////////////////////////////////////
///////////////////////// REST ENDPOINTS START HERE ///////////////////////////
///////////////////////////////////////////////////////////////////////////////

// Register and enroll user
app.post('/enroll', async function (req, res) {
    const username = req.body.username;
    const orgName = req.body.orgName;

    logger.debug(`Endpoint : /enroll, Username: ${username}, OrgName: ${orgName}`);

    let uuid = EnrollmentDecision.init(username);

    // TODO: Change this implementation of sequential processing to correct one.
    async function handle(req, res) {
        if (!EnrollmentDecision.acquire(username, uuid)) {
            setTimeout(function() { handle(req, res); }, 1000);
            return;
        }

        if (!username) {
            res.statusCode = 400;
            EnrollmentDecision.clear(username, uuid);
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
            EnrollmentDecision.clear(username, uuid);
            res.json(adminEnrollResult);
            return;
        }

        const userEnrollResult = await userActions.enroll(username);
        if (userEnrollResult.success) {
            EnrollmentDecision.clear(username, uuid);
            res.json({token: token});
        } else {
            res.statusCode = 400;
            EnrollmentDecision.clear(username, uuid);
            res.json(userEnrollResult)
        }
    }

    handle(req, res);
});

// Invoke transaction on custom chaincode.
app.post('/invoke/:chaincode', async function (req, res) {
    const isObject = req.body.isObject || false;
    const chaincode = req.params.chaincode;
    const fcn = req.body.fcn;
    const args = isObject ? JSON.stringify(req.body.args) : req.body.args;

    logger.debug(`Invoke function: ${fcn} with arguments: ${args}`);

    if (!chaincode) {
      res.statusCode = 400;
      res.json(getErrorMessage('\'chaincode\''));
      return;
    }

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

    res.send(await chaincodeActions.invokeCode(req.username, chaincode, fcn, args, isObject));
});

// Invoke transaction on chaincode
app.post('/invoke', async function (req, res) {
    const isObject = req.body.isObject || false;
    const fcn = req.body.fcn;
    const args = isObject ? JSON.stringify(req.body.args) : req.body.args;

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

    let message = await chaincodeActions.invoke(req.username, fcn, args, isObject);
    res.send(message);
});

// Query transaction on custom chaincode.
app.post('/query/:chaincode', async function (req, res) {
    const isObject = req.body.isObject || false;
    const chaincode = req.params.chaincode;
    const fcn = req.body.fcn;
    const args = isObject ? JSON.stringify(req.body.args) : req.body.args;

    logger.debug(`Query function: ${fcn} with arguments: ${args}`);

    if (!chaincode) {
      res.statusCode = 400;
      res.json(getErrorMessage('\'chaincode\''));
      return;
    }

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

    res.send(await chaincodeActions.queryCode(req.username, chaincode, fcn, args, isObject));
});

// Query transaction on chaincode
app.post('/query', async function (req, res) {
    const isObject = req.body.isObject || false;
    const fcn = req.body.fcn;
    const args = isObject ? JSON.stringify(req.body.args) : req.body.args;

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

    let message = await chaincodeActions.query(req.username, fcn, args, isObject);
    res.send(message);
});
