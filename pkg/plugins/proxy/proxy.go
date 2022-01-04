package proxy

import (
	"net/http"
	"rpc-gateway/pkg/plugins"
)

type Request struct {
	Header    http.Header
	Method    string
	To        string
	Query     interface{}
	TimeOut   int
	CacheTime int
}

type Plugin plugins.Plugin
