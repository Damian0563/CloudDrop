.PHONY: all build clean install test run

BINARY_NAME=clouddrop
GO_FILES=$(shell find . -name '*.go' -not -path './authority/*')
INSTALL_DIR=/usr/local/bin
TARGET=$(INSTALL_DIR)/$(BINARY_NAME)
CREDENTIALS_PATH?=$(HOME)/.clouddrop/credentials.json
BUCKET_NAME?=clouddrop
AUTHORITY?=http://34.118.126.125:8000
LDFLAGS=-ldflags="-s -w \
	-X main.defaultCredentialsPath=$(CREDENTIALS_PATH) \
	-X main.defaultBucketName=$(BUCKET_NAME) \
	-X main.defaultAuthority=$(AUTHORITY)"

all:
	@echo "Run make install to install the binary"
install:
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(GO_FILES)
	@sudo cp $(BINARY_NAME) $(TARGET)
uninstall:
	@sudo rm -f $(TARGET)

