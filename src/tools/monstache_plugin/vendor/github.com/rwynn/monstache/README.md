# monstache
a go daemon that syncs mongodb to elasticsearch in realtime

[![Monstache CI](https://github.com/rwynn/monstache/workflows/Monstache%20CI/badge.svg?branch=rel6)](https://github.com/rwynn/monstache/actions?query=branch%3Arel6)
[![Go Report Card](https://goreportcard.com/badge/github.com/rwynn/monstache)](https://goreportcard.com/report/github.com/rwynn/monstache)

### Version 6

This version of monstache is designed for MongoDB 3.6+ and Elasticsearch 7.0+.  It uses the official MongoDB
golang driver and the community supported Elasticsearch driver from olivere.

Some of the monstache settings related to MongoDB have been removed in this version as they are now supported in the 
[connection string](https://github.com/mongodb/mongo-go-driver/blob/v1.0.0/x/network/connstring/connstring.go)

### Changes from previous versions

Monstache now defaults to use change streams instead of tailing the oplog for changes.  Without any configuration
monstache watches the entire MongoDB deployment.  You can specify specific namespaces to watch by setting the option
`change-stream-namespaces` to an array of strings.

The interface for golang plugins has changed due to the switch to the new driver. Previously the API exposed
a `Session` field typed as a `*mgo.Session`.  Now that has been replaced with a `MongoClient` field which has the type
`*mongo.Client`. 

See the MongoDB go driver docs for details on how to use this client.
