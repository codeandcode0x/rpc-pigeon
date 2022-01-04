package proxy

import (
	"fmt"
	"net"
	logging "rpc-gateway/pkg/core/log"
	grpcPool "rpc-gateway/pkg/plugins/pool/grpc"
	"strings"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var grpcViper *viper.Viper

func init() {
	// get config
	grpcViper = viper.New()
	grpcViper.SetConfigName("PoolConfig")
	grpcViper.SetConfigType("yaml")
	grpcViper.AddConfigPath("config/")
	err := grpcViper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error get pool config file: %s", err))
	}
}

// rpc server
func (plugin *Plugin) GRPCServer() {
	// init config
	rvRoot := grpcViper.AllSettings()["pool"]
	setting := rvRoot.(map[string]interface{})["setting"].(map[string]interface{})
	// whether pool enabled
	if !setting[strings.ToLower("ENABLED")].(bool) {
		return
	}
	// init grpc pool
	grpcPool.InitGrpcPool()
	// engines map
	engines := rvRoot.(map[string]interface{})["engine"]
	for _, v := range engines.([]interface{}) {
		vMap := v.(map[interface{}]interface{})
		// get pool enabl
		if !vMap["POOL_ENABLED"].(bool) {
			continue
		}
		// run grpc server
		go plugin.grpcServer(setting[strings.ToLower("GATEWAY_PROXY_ADDR")].(string)+":"+vMap["GATEWAY_PROXY_PORT"].(string), vMap["ENGINE_NAME"].(string))
	}
}

// asr grpc server
func (plugin *Plugin) grpcServer(addr, serverName string) {
	logging.Log.Info(serverName, " gRPC Server start ...")
	// get gRPC port
	defer func() {
		plugin.Status <- false
	}()

	// get gRPC port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logging.Log.Errorf("failed to listen: %v", err)
		return
	}
	// grpc new server
	srv := grpc.NewServer(grpc.CustomCodec(grpcPool.Codec()),
		grpc.UnknownServiceHandler(grpcPool.TransparentHandler(grpcPool.GrpcProxyTransport)))
	// register service
	grpcPool.RegisterService(srv, grpcPool.GrpcProxyTransport,
		"PingEmpty",
		"Ping",
		"PingError",
		"PingList",
	)
	reflection.Register(srv)
	// start ser listen
	err = srv.Serve(lis)
	if err != nil {
		logging.Log.Errorf("failed to serve: %v", err)
		return
	}
}
