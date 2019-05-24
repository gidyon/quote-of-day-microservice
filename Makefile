SERVER_BUILD_NAME := "quote-app"
SERVER_PKG_BUILD := app.go

build_server_prod: ## Build a production binary for server
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -a -installsuffix cgo -ldflags '-s' -v -o $(SERVER_BUILD_NAME) $(SERVER_PKG_BUILD)