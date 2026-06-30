.PHONY: all build clean install test run

BINARY_NAME=clouddrop
GO_FILES=$(shell find . -name '*.go' -not -path './authority/*')
INSTALL_DIR=/usr/local/bin
TARGET=$(INSTALL_DIR)/$(BINARY_NAME)
AUTHORITY=<url_to_authority server>
BUCKET_NAME=<bucket_name to store data within gcp>
SECRET=<arbitrary secret to encrypt data in network communication>
GOOGLE_JSON=$(shell base64 -w0 authority/credentials.json)
LDFLAGS=-ldflags="-s -w \
	-X main.defaultGoogleJson=$(GOOGLE_JSON) \
	-X main.defaultSecret=$(SECRET) \
	-X main.defaultBucketName=$(BUCKET_NAME) \
	-X main.defaultAuthority=$(AUTHORITY)"

all:
	echo "Run make install to install the binary"
install:
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(GO_FILES)
	@sudo cp $(BINARY_NAME) $(TARGET)
uninstall:
	@sudo rm -f $(TARGET)

release:
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(GO_FILES)
	@tar -czvf clouddrop-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m).tar.gz $(BINARY_NAME)
