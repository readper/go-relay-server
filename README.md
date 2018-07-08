# go-relay-server
telnet to http relay server

# usage
go run main.go [--host] [--port] [--limit-per-second]
  * --host setting listen host, keep empty to listens on all available unicast and anycast IP addresses of the local system.
  * --port setting listen port, default use port 23
  * --limit-per-second setting rate limit, when limit is exceeded, outgoing request will be delayed. 