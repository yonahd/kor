# KOR Backend

Kor Backend is an API wrapper designed to expose Kor.

A tool that identifies unused resources in Kubernetes clusters - over HTTPS.

This backend allows users to access Kor's functionality via RESTful API calls,

making it easier to integrate into web services, dashboards, and automation pipelines.

## Swagger
```swag init```


## Docker
#### Generate SSL certs
```bash
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 36500 -nodes -subj "/C=US/ST=California/L=San Francisco/O=MyCompany/OU=IT/CN=localhost"
```

```bash
docker buildx build -t kor-backend:latest .
```

## Development
```bash
export NO_AUTH=true
go run .
```