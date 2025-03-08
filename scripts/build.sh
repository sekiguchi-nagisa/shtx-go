#!/bin/sh

SCRIPT_DIR=$(cd $(dirname $0); pwd)

cd "$SCRIPT_DIR/../"  # move to project top

GOTOOLCHAIN=auto go build -buildvcs=false -v ./...