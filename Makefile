.PHONY: build test bundle clean

GO ?= go
PLUGIN_ID ?= com.custom.guest-isolation
VERSION ?= 0.1.0
DIST_DIR ?= server/dist

build:
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -trimpath -o $(DIST_DIR)/plugin-linux-amd64 ./server
	GOOS=linux GOARCH=arm64 $(GO) build -trimpath -o $(DIST_DIR)/plugin-linux-arm64 ./server
	GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -o $(DIST_DIR)/plugin-darwin-amd64 ./server
	GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -o $(DIST_DIR)/plugin-darwin-arm64 ./server
	GOOS=windows GOARCH=amd64 $(GO) build -trimpath -o $(DIST_DIR)/plugin-windows-amd64.exe ./server

test:
	$(GO) test -v ./server/...

bundle: build
	rm -f $(PLUGIN_ID)-*.tar.gz
	tar -czf $(PLUGIN_ID)-$(VERSION).tar.gz plugin.json $(DIST_DIR)

deploy: build
	mkdir -p ../../volumes/app/mattermost/plugins/$(PLUGIN_ID)/server/dist
	cp plugin.json ../../volumes/app/mattermost/plugins/$(PLUGIN_ID)/
	cp $(DIST_DIR)/* ../../volumes/app/mattermost/plugins/$(PLUGIN_ID)/server/dist/

clean:
	rm -rf $(DIST_DIR) dist *.tar.gz
