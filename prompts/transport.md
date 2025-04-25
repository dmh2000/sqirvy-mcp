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

I think the code in sse.go is too complex. I want to completely refactor it. 
1. remove all existing code in see.go
2. create a function that creates a server that listens for the SSE GET request from the client. it shold starts the server in a go routine, and forward any data it receives through a channel to a function that implements a reader. return the reader to the caller. do not do anything for the post request yet.

sse.go line 52: should use a simple channel send/receive instead of select with a single case 

create new file pkg/transport/sse_writer.go,  in this file, implement a function  NewSSEWriter. this function will receive an ip, a port and a path. it will return a custom io.writer. this io.writer will take its input data and send an SSE POST request to the specified endpoint.