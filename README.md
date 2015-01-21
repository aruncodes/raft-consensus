## memcached-golang
A simple Memcached clone written in golang


####Introduction

This is a version controlled key-value pair store written in GO. It stores ASCII keys and values along with their expiry time and version. A value is updated only if the version is matched. It runs as a TCP server in port 9000 (configurable). It is written using Go's built-in concurrent abilities. It is packed with testing code also.

####How to install

You can install the server by executing the following commands.
*You need to have golang installed.*
```shell
go get github.com/aruncodes/memcached-golang
go install github.com/aruncodes/memcached-golang
./bin/server
```

####How to use
After the server has started, you need to establish a TCP connection with it for any commands which follows.
The TCP server runs on port 9000.

####Commands

#####1. SET
Set command stores a key value pair into datastore. It requires expiry time in seconds and length of data. It is a two line command with data in its second line.

Syntax:
```
	set <key_name> <expiry_time> <num_bytes> [noreply] \r\n
	<value>
```
```<key_name> ```: Name of key (ASCII)

```<expiry_time> ```: Expiry of the value in seconds. 0 means no expiry.

```<num_bytes>	```: Size of data in bytes

```<value>	```:Actual data (as next line)

```[noreply]``` : (Optional) If you don't need response from server

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
	cas <key_name> <expiry_time> <version> <num_bytes> [noreply] \r\n
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

####Expiry Handler
The server includes an expiry handler which removes a key value pair when its expiry time is reached. Expiry time is calculated as no. of seconds provided when the key is set.

####How to test server
The server is built with go's testing mechanism. You can test the server by executing
```shell
go test github.com/aruncodes/memcached-golang
```