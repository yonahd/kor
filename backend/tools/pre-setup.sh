#TODO: place static files in dockerfile
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
go run server.go