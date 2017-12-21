#!/usr/bin/env bash
rm dist/*
env GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/migo-v1-amd64.exe
env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/migo-v1-amd64
