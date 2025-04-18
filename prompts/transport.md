in file pkg/transport/transport.go, create:

1. a function that receives three arguments, a standard go byte 'reader', a  buffered channel and a logge, and returns an error if any
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

