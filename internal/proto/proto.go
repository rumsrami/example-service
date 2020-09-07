// Package proto ...
//go:generate webrpc-gen -schema=chat.ridl -target=go -pkg=proto -server -client -out=./chat.gen.go
//go:generate webrpc-gen -schema=chat.ridl -target=ts -pkg=proto -client -out=./chat.gen.ts
package proto
