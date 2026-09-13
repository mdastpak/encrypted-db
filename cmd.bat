:=== cmd.bat ===

@echo off

go mod tidy
swag init -g ./main.go
go run ./cmd/server/main.go

:===
