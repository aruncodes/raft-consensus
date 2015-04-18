## raft-golang
A partial implementation of RAFT consensus algorithm. Work still in progress.


####Introduction
The current version does not deal with leader election and thereby does not perform certain functions like heartbeat sending etc. This version only deals with log replication. It reads in the cluster configuration from a json file and assumes that the first server configuration is for the leader. It returns a redirect message to the client if it tries to contact any other server which is not the leader.


####How to install

You can install the server by executing the following commands.
*You need to have golang installed.*
```shell
go get github.com/aruncodes/cs733/assignment2/kvstore
go install github.com/aruncodes/cs733/assignment2/kvstore
./bin/kvstore <server-id>
```


####How to use
After the server has started, you need to establish a TCP connection with it for any commands which follows.
The TCP server runs on port 9000.


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
The server is built with go's testing mechanism. You can test the server by executing
```shell
go install github.com/aruncodes/cs733/assignment2/kvstore
go test github.com/aruncodes/cs733/assignment2/kvstore
```

####Testing
This version is just an extension to the previously built memcached kvstore. Thus the basic underlying code for a client handler and a backend kvhandler is similar. As a result test cases for assignment 1 are not included in the testing functions of this version.

Testing functionality is added for the newly built log replication module which will also eventually implement raft.  
First test includes spawning a server which is not a leader. If a client tries to make contact to it a REDIRECT error message must be sent to the client.  
Second test includes spawning 2 servers one of which is the leader. The client contacts the leader but the leader does not have any majority as of now. After some time we spawn a new follower server and now the leader gets a majority and the client request is passed through, processed, and results sent back.