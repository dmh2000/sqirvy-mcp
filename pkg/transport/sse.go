package transport

func SseSender(get_addr string,port int, post_addr string,port ) (get <-chan []byte, post chan<-[]byte) {

	get = make(<-chan []byte, 1)
	post = make(chan<- []byte, 1)

	// add HTTP Server here that listens for a  GET request


	// add HTTP Server here that listens for POST requests

	return get,post
}
