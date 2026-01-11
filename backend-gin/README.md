git pull

# Install required dependencies
```bash
go get github.com/golang-jwt/jwt/v5
go get github.com/lib/pq 
go get golang.org/x/crypto/bcrypt
go get github.com/gin-gonic/gin
go get golang.org/x/time/rate
```
# Run your server
```go
go run .
```
##***TROUBLESHOOTING***
-must have .env since i'll not push
```shell
touch backend-gin/.env
```
**to generate JWT_SECRET**
```shell
openssl rand -base64 64
```

# Database connection and creation
**
# Api testing
```shell
curl 
```