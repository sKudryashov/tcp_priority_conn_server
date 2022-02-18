### StackServer instructions:

#### Technologies used: GoLang, Docker

To start server just go the root and type “make takeoff”, if it doesn't work, ensure make tool is installed.
To restart it - push on port 8081 “rel” command to the socket, in stack-test.rb it is “push_reload” action. Each test prepended with push_reload call to reset it’s state (as mentioned in the spec) except for test_single_request because it is used in different tests and may reset the connection in the middle. 

IF something is not working after socket server restart command - just CTRL+C in the container frontend and “make takeoff” once again but it’s unlikely - test “test_multiple_reload” was added to the test suite to ensure works normally.

The app is built with enabled race detector, this is not an option for the prod of course since it slows down performance, but in our case is to ensure there are no race conditions. The profiler on port 8090 commented and left just in case. 

If there is too much logging - you may reduce logging level change LOG_LEVEL: debug in docker-compose file to “info” or “error”. The stack-related logs are on the info level.

Also some unit tests are available, they are in *_test.go files. 

### Socket Server Description
A server that manages a LIFO stack, supporting push and pop
operations. The server listens for requests from clients connecting over TCP
on port 8080. The server should respond to the request and then close the
connection.

A push request pushes the given payload onto the stack. However, the stack
can have no more than 100 items on it. Push requests for a full stack should
block until stack space becomes available. (This is similar to how HTTP
long-polling works.)

A pop request returns the top item from the stack to the client. If the
stack is empty, the pop request should block until an item becomes available
on the stack. (This, again, is similar to how HTTP long-polling works.)

Both push and pop requests should be served (and their associated stack
operations performed) in the order in which they arrive fully. Note that
this is not necessarily the order in which the server accepts TCP
connections: some clients may be very slow to write their requests. Clients
that connect later may 'overtake' slower clients that are still writing
their request. Those fast clients should get their response before the slow
clients.

The server should not have to juggle more than 100 clients simultaneously.
Additional client connections should be rejected by sending a single byte
response indicating busy-state and then immediately disconnected. (Strictly
speaking, this means that there is a very brief period during which the
server is handling more than 100 simultaneous connections--but only long
enough to dismiss those additional clients.)

However, there is an exception to this rule. To prevent deadlock (eg, 100
pop requests all waiting for a push request that is always rejected) the
server must free up resources under specific conditions. If the server is
already handling 100 connections and a new client tries to connect, it must
disconnect the oldest client, provided their connection is older than 10
seconds. The server should only disconnect an old client when provoked by a
new incoming client connection that would otherwise have to be rejected. It
should not just disconnect old clients for no reason.

A push request format is as follows. The first byte is the header. The rest
of the request is the payload. The most significant bit in the header byte
is 0; the 7 remaining bits are the length of the payload, in bytes. (As
such, the minimum size for a push request is 2 bytes: 1 header byte and 1
payload byte. The maximum size for a push request is 128 bytes: 1 header
byte and 127 payload bytes.)

The format of a pop request is a single byte with the most significant bit
set to 1. The rest of the byte is ignored.

The format of a push response is 1 byte, all zeros.

The format of a pop response is 1 header byte, with the most significant bit
set to 0 and the rest of the payload indicating the payload size. The rest
of the response is the payload indicated number of bytes of payload.

The format of a busy-state response is 1 byte, value 0xFF.


You have to assume little-endian byte ordering, the default on x86
architectures. Bytes are sent in network order.



Another simplifying assumption you get to make is that once you pop
something off the stack, that's it. You don't have to put it back on the
stack if it turns out you can't write the response to the client (for
example, if they disconnected).

You might find it useful to implement a listening socket on, say, port 8081
that prints out the state of the stack (and other debugging info you might
need) and then closes the connection. This is optional, of course.

This test suite assumes the server starts out with an empty stack. If this
test suite crashes you should restart the server before running the test
suite again. You are welcome to modify the server to implement a reset code
(probably by interpreting the otherwise ignored remaining 7 bits in a pop
request). Alternatively, have the server listen on another port over which
you can send reset commands in setup or teardown.

The test suite will cleanly close all of its connections when the teardown
runs after each test. Your code will need to account for, and cleanup closed
connections. To do this in a POSIX world, you must attempt to read from the
rocket; if your read call returns 0, then the connection has been closed.
See the man page on recv:
http://man7.org/linux/man-pages/man2/recv.2.html#RETURN_VALUE

You conveniently--and falsely--get to assume (and need to ensure) that
sockets are always closed completely. In other words, the client and the
server will not close the socket only for reading or writing, while keeping
the other half open. Phrased differently, if the return value of a read()
call on a socket indicates that the connection is closed, you can assume the
connection is also closed for writing. The inverse is true, also.

This test suite is representative, but not comprehensive. We encourage you
to write more tests, but this is not required.

You are welcome to write this in the language of your choice! You should
pick a language that you are very comfortable with; don't try an implemention
in CuttingEdgeLanguage2000 just because you think it will impress us.
We'd rather see a clean solution in BoringOldLanguage. You may use whichever
libraries you wish, but please provide clear installation instructions.