@echo off
if not exist dist mkdir dist
go build -o dist\xyz-mcp.exe main.go 