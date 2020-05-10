#!/bin/bash

export GOPATH="$GOPATH:$(pwd)"
export GOOS=windows
go env
