in file pkg/transport/transport.go, create:

1. a function that receives three arguments, a standard go byte 'reader', a  buffered channel and a logger, and returns an error if any
2. the function will enter a continuous loop, reading messages and sending them to the channel.
3. a messages is a stream of bytes that is delimited by a '\n' newline.
4. the function will read  bytes until a newline is encountered, then the function will do the following
5. trim whitespace from front and back of the message. 
6. if the message is empty, log itand continue reading
7. test the message to see if it looks like a valid JSON message
8. if the message is not valid JSON, log it, discard it and continue reading
9. if the message is valid JSON, send it to the channel
10. if the channel is not full, send the message to the channel
11. if the channel is full, wait one second and retry the send
12. if the channel is closed, return an error indicating the channel is closed
13. if the reader is closed, return an error, indicating the reader is closed
14. if the reader is not closed, continue reading


in new file pkg/transport/sse.go, create a function, NewSSEReader, this function will initialize the input side of a model context protocol server side event reader. The function will be given an IP address, a port and a path, and it will initialize the SSE protocol by sending the Get request  and return an io.reader connected to the stream and an error.

in file pkg/transport/sse.go, add a function, NewSSEWriter that implements the server to client side of the SSE transport. The function that is given an ip address, a port and a path. This function will create a function closure over the ip address, port and path, and return that function closure. the function closure will return an io.writer function that when called, will send a Post request to the ip, port and path, with the mime type application/json.


you are creating the server endpoints that implements an mcp server using the Server Side Events protocol.
add the following to the file pkg/transport/sse.go:
- The first endpoint is an http server accepting the text/event-stream mime type. 
  - the server is created with a function NewSseServer that gets an ip address, a port and path as input 
- The second endpoint is an http server arguments. it returns two buffered channels, named 'output' and 'input'
  - the NewSseServer does 
    - create an input only channel 'get' that receives bytes slices
    - create an output only channel 'post' that sends byte slices 
    - in a goroutine, start an HTTP server 
        - this server waits for a connection from the mcp client. 
        - when the client connects, it sends an HTTP GET request with the content-type text/event-stream
        - when the server receives the GET request, it responds with an event like this:
            event: endpoint
            data: /messages?session_id=(unique session id)
   - after that initialization, this server starts a loop where it receives json-rpc messages on the 'input' channel. when it receives a message on the 'get' channel, it sends it out the on the HTTP connection. it will continue this loop until the http connection is closed
   - in a second goroutine, start another HTTP server listening on the /messages endpoint
   - the server receives http messages with 



