# mcache

This is project satisfies Slack's interview assignment to implement a subset of a memcache server. See /assignment.htm
for details. To summarize, this is an implementation of a memcache server that speaks the memcache text protocol.
It supports the set, get, gets, delete, and cas commands, but without any expiration logic.

## Getting started

### Building

To build the project, you will need the golang toolchain. It can downloaded [here](http://golang.org). Once that is
installed, do the following:

1. Obtain a copy of this git repo, either by git clone or by unpacking a copy of this repo. Name it "mcache"

2. Create a directory structure to serve as a GOPATH for building:
    ```$ mkdir -p gopath/src/github.com/tshprecher/```

3. Assign the GOPATH env var:
    ```$ export GOPATH=$PWD/gopath```

4. Move the directory from step 1 into the gopath:
    ```$ mv mcache gopath/src/github.com/tshprecher/mcache```

5. Build the binary:
    ```$ go install github.com/tshprecher/mcache/...```

6. The `mcache` binary should now be contained in $GOPATH/bin/

### Testing

To run all the tests, unit and integration, make sure to complete steps 1-4 above and run:

```$ go test github.com/tshprecher/mcache/...```

Note that the integration tests open memcache servers on port 11210 and 11209, so please make sure nothing is open on those ports
when testing.

### Running

To start the server on the standard memcache port 11211, run `$ mcache -stderrthreshold=0`

There are four parameters you can set upon startup:
* `port`: the port to listen on (default: 11211)
* `cap`: the total capacity in bytes to allow for storage, including the space for keys (default: 1GB)
* `timeout`: the time in seconds a session is allowed to be idle before being closed by the server to free up resources (default: 5)
* `max_val_size`: an explicit limitation in bytes on the size a value can be so that clients cannot overload the server with data (default: 0, indicating no limit)


### Design

#### Overview

The project has sufficient documentation comments, but here is the summary of the server's components:

The `StorageEngine` interface owns the core logic of setting and retrieving values into and out of memory. At the moment,
there is only one implementation, the `SimpleStorageEngine`. The `SimpleStorageEngine` is a naive approach backed by a simple
golang map and handles concurrency by locking on all operations. This is a surely a bottleneck for performance, as a striped
locking implementation should outperform. While I didn't implement striped locking storage engine for the sake of saving time,
implementing one as `StorageEngine` and plugging it into the rest of the code should not be an onerous task since the rest of the
codebase works against the `StorageEngine` interface.

The `EvictionPolicy` interface defines an interface that `StorageEngine`s use to evict keys when necessary. It's an interface
because there should be a few implementations that satisfy the problem: LRU, MRU, and LFU, for example. I chose only to implement
and LRU eviction policy for the sake of time, so the server currently always evicts the least recently used keys.

The `MessageBuffer` interface defines Read() and Write() operations for unpacked requests and responses. This wraps around
the tcp connection for serializing and deserializing messages to and from the wire. There's currently only one implementation,
the `TextProtocolMessageBuffer`, because right now this only supports the memcache text protocol.

The `TextSession` struct wraps a `MessageBuffer` and implements the `Serve()` method that reads a command from the `MessageBuffer`,
implements the protocol's business logic by talking to the `StorageEngine`, and writes a response back to the client
via the `MessageBuffer`.

Finally, the `Server` struct listens on a port for a connection. Each new connection is wrapped around a `TextSession`
and handed off to a goroutine that constantly polls `TextSession.Serve()` to serve the client while the session is alive.
This means there is one goroutine per connection. Again, this may not be the most efficient implemenation, but it allows
for multiple concurrent sessions. Goroutines are cheap. Thousands of them can be running concurrently, but eventually
the overhead of the goroutine scheduler could impede performance of the server and a system where there are fixed number
of worker goroutines could be adopted.

#### Testing

There are unit tests for the `StorageEngine`, `EvictionPolicy`, and the `MessageBuffer`. For the `Server` and `TextSession`,
 I traded unit tests for integration tests for the sake of time, and I intended to have them anyway. There are two integration
test suites, protocol and client. The protocol integration tests spin up a server, write bytes to a tcp connection, and
checks received bytes against expectations. This is allows for testing of explicit error messages and other protocol details.
It helped when implementing the "noreply" flag for each command, for example. Finally, the client integration tests spin up
a server and talk to it using Brad Fitzpatrick's official memcached go [client](https://github.com/bradfitz/gomemcache).
Instructions for running the tests are above.

### Discussion

#### Performance Expectations

The performance of this server will obviously depend on the workload (level of concurrency, size of values, etc) and hardware.
That's not to say that all components are currently written to be highly performant. Without load testing on the proper hardware,
it's hard to give details. On the proper hardware, I should expect this server to handle thousands, if not 10s of thousands
of concurrent connections, even with one goroutine per connection. The initial bottleneck will almost surely be the level
of true concurrency because of the coarse grained locking of the `SimpleStorageEngine`. This would be the first thing I would
change by perhaps implementing a `StripedLockingStorageEngine`. Others possible improvements include:

* Changing the `MessageBuffer` to work with buffered readers and writers to reduce the number of syscalls made.
* Find out the limitations of one goroutine per connection and perhaps separate the tasks of detecting when bytes
are available to be read from the wire from the task of serving the request. This could allow for two pools of goroutines,
one for servicing requests and one for detecting network traffic, to slow down the rate of goroutine startup and teardown
and the overhead in scheduling one goroutine per connection.

#### Monitoring

Monitoring really depends on the environment in which this runs. I've worked in environments that use a stats daemon to
collect stats written to it by servers, but I've also seen raw log collection and uploading to services like Scalyr for
analysis. The server currently uses glog for logging, so the latter case could be easily adopted after adding log lines
for the pertinents stats. As for the latter, some module that talks to the stats daemon would have to be used.

But regarding monitoring, the most important question to ask is what should we monitor? I would keep an eye on the

1. Latency distributions of the storage engine, segmented by operation (set/get) and the size of the data being
sent and received (100bytes, 1KB, 100KB, 1MB, etc.)
2. Latency distributions of the network overhead for each request, segmented by operation and the size of the data being sent and
received.
3. Number of concurrent connections allowable by the networking portion of the implementation, in this case strongly
related to the number of running goroutines
4. Latency distributions clients sees to get a value, segmented by size. How long does it take from the first byte read to the
last by written for a request?
4. eviction rates to see if we're using the proper policy for the workload

#### Possible quirk(s)

The current implementation does not buffer reads or writes to the TCP connection. Currently it's all or nothing for writing
a response back to the client, which means that if the server tries to write a response that's N bytes and only N-1 bytes
are written, the server treats it as a server error and closes the connection. Reads, however, do support bursty traffic
from the client.

### Final thoughts

I had a good enough time completing this project that I may go back and fill in some of the gaps in the protocol. My
goal for this project was to get it to a place where the individual components make sense and could easily be iterated
upon. Want to implement the binary protocol? Just implement a `BinaryProtocolMessageBuffer`. Want to implement a new
eviction policy? Create a new implementation of `EvictionPolicy`. The same goes for the `StorageEngine`.
