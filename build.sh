#!/bin/bash

GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -trimpath -ldflags '-w -s'
