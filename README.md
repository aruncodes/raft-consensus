## raft-golang
A distributed Memcached clone (KeyValue store) written in golang backed by RAFT consensus algorithm.

####Introduction
This project is done in 4 parts.

##### 1) Key Value store (not distributed)
See assignment1 folder

##### 2) Key Value store with consensus among cluster.
Bare bones distributed, single fixed leader,heart beats, majority voting, no log replication etc
See assignment2 folder

##### 3) Internally Distributed KV Store 
Complete distributed RAFT inside single binary. KV store, leader election, heart beats, log replication, store state to disk, internally distributed, uses channels instead of RPCs. Can have any number of servers inside as separate go routines.
See assignment3 folder

##### 4) Distributed KV Store
Completely distributed, backed by RAFT. Supports KV store, crash tolerant, heart beats, leader election, log replication, store state to disk, RPC communication.
See assignment4 folder