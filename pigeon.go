//main
package main

import (
	"rpc-gateway/pkg/plugins"
	"rpc-gateway/pkg/plugins/httpserver/util"
	"rpc-gateway/pkg/plugins/metrics"
	"rpc-gateway/pkg/plugins/proxy"
)

// main
func main() {
	util.InitConfig()
	var httpPlugin, gRPCPlugin, vsPlugin proxy.Plugin
	var metricsPlugin metrics.Plugin
	// register gRPC 、HTTP Server
	gRPCPluginChan := registerPlugins(gRPCPlugin.GRPCServer)
	httpPluginChan := registerPlugins(httpPlugin.HttpServer)
	// register Metrics Data Server
	metricsPluginChan := registerPlugins(metricsPlugin.ShowMetrics)
	// register VS
	kvsPluginChan := registerPlugins(vsPlugin.KvsServer)

	// return status chan
	<-kvsPluginChan
	<-gRPCPluginChan
	<-httpPluginChan
	<-metricsPluginChan
}

// register plugins
func registerPlugins(factory func()) chan bool {
	var rpcProxy plugins.Plugin
	rpcProxy.Factory = factory
	rpcProxy.Status = make(chan bool)
	go rpcProxy.Factory()
	return rpcProxy.Status
}
