#Generate token
cd kor/backend/tools
export KOR_API_SECRET=abcd12; export KOR_API_TOKEN=$(go run generate_token.go)
#In case no auth needed
export NO_AUTH=true
#TODO: place static files in dockerfile
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
go run main.go

### SWAGGER ####
# Install swag cli
go install github.com/swaggo/swag/cmd/swag@latest
# Regenerate swagger
swag init
#Swagger ui
https://localhost:8080/swagger/index.html
