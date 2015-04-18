## raft-golang
A distributed implementation of Key Value store like memcached which is backed by RAFT consensus algorithm. It is written using golang 1.4.


####Introduction
RAFT ensures consensus across servers. If a client gets confirmation from the cluster, it will be available even if only majority of servers are alive. For any transaction to succeed, it should be approved by majority of servers. Eg: If there are 5 live servers, then at-least 3 should approve a transaction.  Once confirmed, the value stays synchronized in all servers. 

There will always be a leader elected among the cluster. If the client needs to do any transaction, it must communicate with leader. If the server connected is not a leader, it will respond a REDIRECT message with leader id which can be used by the client to connect to leader. 

There can be any number of servers as specified by the config.json file. The client ports, log ports, server id etc are specified in config file. Servers use go channels to communicate with each other.

This version only have internal representaion of a server. It doesn't use RPCs. 

####How to install

You can install the server by executing the following commands.
*You need to have golang installed.*
```shell
go get github.com/aruncodes/cs733/assignment3/kvstore
go install github.com/aruncodes/cs733/assignment3/kvstore
./bin/kvstore
```


####How to communicate
After the servers has started, you need to establish a TCP connection with it for any commands which follows.
The TCP server runs on port (9000 + server-id). (Can be changed in config file) If you receive redirect messages for commands, you need to establish a new connection to leader which is specified in the redirect message.


####Commands

#####1. SET
Set command stores a key value pair into datastore. It requires expiry time in seconds and length of data. It is a two line command with data in its second line.

Syntax:
```
	set <key_name> <expiry_time> <num_bytes> \r\n
	<value>
```
```<key_name> ```: Name of key (ASCII)

```<expiry_time> ```: Expiry of the value in seconds. 0 means no expiry.

```<num_bytes>	```: Size of data in bytes

```<value>	```:Actual data (as next line)

**Response:**

Success : 
``` OK <version>```

Failures :

```ERR_CMD_ERR``` : Error in your command or arguments.

```ERR_VERION``` : Trying to overwrite existing key.

#####2. GET
Get command allows to get the value of the key provided if it exists.

Syntax:
```
	get <key_name>\r\n
```
```<key_name> ```: Name of key (ASCII)

**Response:**

Success : 
``` 
	VALUE <num_bytes> \r\n
	<value> \r\n
```

Failures :

```ERR_CMD_ERR``` : Error in your command or arguments.

```ERR_NOT_FOUND``` : Value doesn't exist anymore. (Expired maybe)

#####3. GETM
Getm command allows to get the value along with meta details of the key provided if it exists.

Syntax:
```
	getm <key_name>\r\n
```
```<key_name> ```: Name of key (ASCII)

**Response:**

Success : 
``` 
	VALUE <version> <expiry_time> <num_bytes> \r\n
	<value> \r\n
```

Failures :

```ERR_CMD_ERR``` : Error in your command or arguments.

```ERR_NOT_FOUND``` : Value doesn't exist anymore. (Expired maybe)

#####4. CAS (Compare and Swap)
Compare and swap command replaces the existing value of a key with new data if the version matches. It also updates the expiry time number of bytes. Other than key name and version, nothing else need to be same as previous value.

Syntax:
```
	cas <key_name> <expiry_time> <version> <num_bytes> \r\n
	<value>
```
```<version> ```: Version number of key you want to swap with.

**Response:**

Success : 
``` OK <new_version>```

Failures :

```ERR_CMD_ERR``` : Error in your command or arguments.

```ERR_VERION``` : Version didn't match.

```ERR_NOT_FOUND``` : Value doesn't exist anymore. (Expired maybe)

#####5. DELETE
Delete command removes a key-value pair from datastore.

Syntax:
```
	delete <key_name>\r\n
```
```<key_name> ```: Name of key (ASCII)

**Response:**

Success : 
``` 
	DELETED \r\n
```

Failures :

```ERR_CMD_ERR``` : Error in your command or arguments.

```ERR_NOT_FOUND``` : Value doesn't exist. (Expired maybe)


####Errors
```ERR_CMD_ERR``` : Unknown command or error in syntax

```ERR_INTERNAL``` : Internal server error

```REDIRECT <leader_id>``` : The server being contacted is not the leader, contact server with leader_id as its id


####Expiry Handler
The server includes an expiry handler which removes a key value pair when its expiry time is reached. Expiry time is calculated as no. of seconds provided when the key is set.


####How to test server
Uses build in testing mechanism of go. It will test the cluster for different features. You can test the server by executing
```shell
go test github.com/aruncodes/cs733/assignment3/kvstore
```
####Testing
Test 1: Kill one leader, check if new leader gets elected.Kill another, check the same.Resurrect previous as follower, check if it sinks in.Resurrect first one as leader, check if it sinks in.

Test 2: Kill 3 servers and make sure no leader gets elected

Test 3: Test if log after append is same accross all servers.Check everyones log with leader's.

Test 4: After crashing and recovering, test if log is replicated after some time.