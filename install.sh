#!/bin/bash

BINARY_NAME=clouddrop
GO_FILES=$(find . -name '*.go' -not -path './authority/*')
INSTALL_DIR=/usr/local/bin
TARGET=${INSTALL_DIR}/${BINARY_NAME}
SHARED_SECRET=${SHARED_SECRET:-changeme}
AUTHORITY=${AUTHORITY:-http://34.118.126.125:8000}
LDFLAGS="-s -w -X main.defaultSharedSecret=${SHARED_SECRET} -X main.defaultAuthority=${AUTHORITY}"

echo "Installing clouddrop..."
go build ${LDFLAGS} -o ${BINARY_NAME} ${GO_FILES}
sudo cp ${BINARY_NAME} ${TARGET}
echo "Installation complete! You can now run 'clouddrop' from the terminal."
