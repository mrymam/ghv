#!/bin/bash
set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="gv"

echo "Building ${BINARY_NAME}..."
go build -o "${BINARY_NAME}" .

echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

echo "Done! Run 'gv' from anywhere."
