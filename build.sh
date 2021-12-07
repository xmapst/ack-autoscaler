#!/usr/bin/env bash
go mod tidy
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o autoscaler .
strip --strip-unneeded autoscaler
upx --lzma autoscaler