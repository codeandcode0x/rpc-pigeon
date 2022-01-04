package grpc

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"rpc-gateway/pkg/core/common"
	logging "rpc-gateway/pkg/core/log"
	"rpc-gateway/pkg/plugins"
	"rpc-gateway/pkg/plugins/vs/kvs"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// pool
type Plugin plugins.Plugin

// config viper
var routerViper *viper.Viper

// pools map
var asrPools, ttsPools = make(map[string]*Pool), make(map[string]*Pool)

// pools options
var asrOptions, ttsOptions Options

// tencants pools map
var tenants map[string]map[string]*Tenant

// failed pool status
var failedPoolStatus bool = false

// engines service port map
var engineSvcPort = make(map[string]string)

// init engine from exist interval time
var enginePoolInitTime = make(map[string]int)

// engine label keys
var engineSvcSelectorKey, asrEngineSvcSelectorVal, ttsEngineSvcSelectorVal string

// enabled engine cluster
var engineClusterEnabled bool = false

// engine node num
var engineClusterNodeNum int

// const
var (
	DialTimeout                          = 5 * time.Second
	BackoffMaxDelay                      = 3 * time.Second
	KeepAliveTime                        = time.Duration(10) * time.Second
	KeepAliveTimeout                     = time.Duration(3) * time.Second
	InitialWindowSize              int32 = 1 << 30
	InitialConnWindowSize          int32 = 1 << 30 // 1073741824
	MaxSendMsgSize                       = 4 << 30 // 4294967296
	MaxRecvMsgSize                       = 4 << 30
	OMPEnabled                           = false
	TenantEnabled                        = false
	GatewayProxyAddr                     = ""
	NetWorkMode                          = 1
	ASRGatewayProxyAddr                  = ""
	TTSGatewayProxyAddr                  = ""
	ENGINE_INIT_POOL_INTERVAL_TIME       = time.Duration(30) * time.Second
)

// init grpc pool
func InitGrpcPool() {
	// get config
	routerViper = viper.New()
	routerViper.SetConfigName("PoolConfig")
	routerViper.SetConfigType("yaml")
	routerViper.AddConfigPath("config/")
	err := routerViper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error get pool config file: %s", err))
	}

	rvRoot := routerViper.AllSettings()["pool"]
	engines := rvRoot.(map[string]interface{})["engine"]
	setting := rvRoot.(map[string]interface{})["setting"].(map[string]interface{})
	// enabled
	if !setting[strings.ToLower("ENABLED")].(bool) {
		return
	}

	DialTimeout = time.Duration(setting[strings.ToLower("DIAL_TIMEOUT")].(int)) * time.Second
	BackoffMaxDelay = time.Duration(setting[strings.ToLower("BACKOFF_MAX_DELAY")].(int)) * time.Second
	KeepAliveTime = time.Duration(setting[strings.ToLower("KEEPALIVE_TIME")].(int)) * time.Second
	KeepAliveTimeout = time.Duration(setting[strings.ToLower("KEEPALIVE_TIMEOUT")].(int)) * time.Second
	OMPEnabled = setting[strings.ToLower("OMP_ENABLED")].(bool)
	TenantEnabled = setting[strings.ToLower("TENANT_ENABLED")].(bool)
	NetWorkMode = setting[strings.ToLower("NETWORK_MODE")].(int)
	GatewayProxyAddr = setting[strings.ToLower("GATEWAY_PROXY_ADDR")].(string)
	engineSvcSelectorKey = setting[strings.ToLower("ENGINE_SERVICE_SELECTOR_KEY")].(string)
	asrEngineSvcSelectorVal = setting[strings.ToLower("ASR_ENGINE_SERVICE_SELECTOR_VALUE")].(string)
	ttsEngineSvcSelectorVal = setting[strings.ToLower("TTS_ENGINE_SERVICE_SELECTOR_VALUE")].(string)
	engineClusterEnabled = setting[strings.ToLower("CLUSTER_ENABLED")].(bool)
	engineClusterNodeNum = setting[strings.ToLower("CLUSTER_NODE_NUM")].(int)
	// engine init
	for _, v := range engines.([]interface{}) {
		vMap := v.(map[interface{}]interface{})
		op := Options{
			Dial:      grpcDial,
			PoolModel: vMap["POOL_MODEL"].(int),
			// MaxIdle:              vMap["ENGINE_GRPC_POOL_SIZE"].(int),
			// MaxActive:            vMap["ENGINE_GRPC_POOL_SIZE"].(int),
			MaxConcurrentStreams: 0,
			Reusable:             vMap["GRPC_REQUEST_REUSABLE"].(bool),
			RequestIdleTime:      vMap["REQUEST_IDLE_TIME"].(int),
			RequestMaxLife:       vMap["REQUEST_MAX_LIFE"].(int),
			RequestTimeOut:       vMap["REQUEST_TIMEOUT"].(int),
			GatewayProxyAddr:     GatewayProxyAddr,
			GatewayProxyPort:     vMap["GATEWAY_PROXY_PORT"].(string),
			PoolStatus:           vMap["POOL_ENABLED"].(bool),
		}

		// get pool enabl
		if !vMap["POOL_ENABLED"].(bool) {
			continue
		}
		// engine server port
		serverPort := vMap["ENGINE_SERVER_PORT"].(string)
		// engine init pool interval time
		poolInitIntervalTime := vMap["ENGINE_POOL_INIT_INTERVAL_TIME"].(int)
		// init server port map
		if strings.ToLower(vMap["ENGINE_NAME"].(string)) == "asr" {
			engineSvcPort["asr"] = serverPort
			enginePoolInitTime["asr"] = poolInitIntervalTime
		} else if strings.ToLower(vMap["ENGINE_NAME"].(string)) == "tts" {
			engineSvcPort["tts"] = serverPort
			enginePoolInitTime["tts"] = poolInitIntervalTime
		}
		// 如果开启 OMP 在线配置则不会读取 config.engine 中的配置
		if OMPEnabled {
			continue
		}
		// engine list
		for _, engine := range vMap["ENGINE_LIST"].([]interface{}) {
			xid := common.GenXid()
			eMap := engine.(map[interface{}]interface{})
			// asr server address
			serverAddr := eMap["SERVER_HOST"].(string) + ":" + serverPort
			// check grpc server status
			checkSerStatus := checkGRPCSerer(serverAddr)
			if !checkSerStatus {
				logging.Log.Error("grpc server connect failed !")
			}
			// setting pool size
			op.MaxIdle = eMap["ENGINE_GRPC_POOL_SIZE"].(int)
			op.MaxActive = eMap["ENGINE_GRPC_POOL_SIZE"].(int)
			// cluster setting
			if engineClusterEnabled {
				if op.MaxIdle%2 == 0 {
					op.MaxIdle = eMap["ENGINE_GRPC_POOL_SIZE"].(int) / engineClusterNodeNum
					op.MaxActive = eMap["ENGINE_GRPC_POOL_SIZE"].(int) / engineClusterNodeNum
				} else {
					op.MaxIdle = eMap["ENGINE_GRPC_POOL_SIZE"].(int)/engineClusterNodeNum + 1
					op.MaxActive = eMap["ENGINE_GRPC_POOL_SIZE"].(int)/engineClusterNodeNum + 1
				}
			}
			// new pool
			p, err := newGrpcPool(serverAddr, op)
			if err != nil {
				log.Fatalf("failed to new pool: %v", err)
			}
			// setting remote addr
			p.poolRemoteAddr = serverAddr
			p.name = xid
			// new pool by engine
			if strings.ToLower(vMap["ENGINE_NAME"].(string)) == "asr" {
				asrOptions = op
				if NetWorkMode == STRICT_NETWORK_MODE {
					ASRGatewayProxyAddr = GatewayProxyAddr + ":" + asrOptions.GatewayProxyPort
				} else if NetWorkMode == GLOBAL_NETWORK_MODE {
					ASRGatewayProxyAddr = asrOptions.GatewayProxyPort
				}
				// asrPools = append(asrPools, p)
				asrPools[xid] = p
			} else if strings.ToLower(vMap["ENGINE_NAME"].(string)) == "tts" {
				ttsOptions = op
				if NetWorkMode == STRICT_NETWORK_MODE {
					TTSGatewayProxyAddr = GatewayProxyAddr + ":" + ttsOptions.GatewayProxyPort
				} else if NetWorkMode == GLOBAL_NETWORK_MODE {
					TTSGatewayProxyAddr = ttsOptions.GatewayProxyPort
				}
				// ttsPools = append(ttsPools, p)
				ttsPools[xid] = p
			}
		}
	} // end init pool
	// init pool from exist engine
	if OMPEnabled {
		logging.Log.Info("starting engine discovering ...")
		go initPoolFromExistEngine(map[string]interface{}{
			"engineType":          "asr",
			"engineSelectorKey":   engineSvcSelectorKey,
			"engineSelectorValue": asrEngineSvcSelectorVal,
		})

		go initPoolFromExistEngine(map[string]interface{}{
			"engineType":          "tts",
			"engineSelectorKey":   engineSvcSelectorKey,
			"engineSelectorValue": ttsEngineSvcSelectorVal,
		})

	} else if TenantEnabled {
		// multi tenant support
		initPoolForTenant()
	}

	// check server pool healthz
	go checkAsrGRPCSererHealthTask()
	go checkTtsGRPCSererHealthTask()
}

// init grpc pool by omp
func InitGrpcPoolByOmp(data map[string]interface{}) {
	rvRoot := routerViper.AllSettings()["pool"]
	engines := rvRoot.(map[string]interface{})["engine"]
	eType := data["engineType"].(string)
	// init pool
	for _, v := range engines.([]interface{}) {
		vMap := v.(map[interface{}]interface{})
		engineType := strings.ToLower(vMap["ENGINE_NAME"].(string))
		if engineType != eType {
			continue
		}
		// server port
		serverPort := vMap["ENGINE_SERVER_PORT"].(string)
		// server addr
		serverAddr := data["serverHost"].(string) + ":" + serverPort
		// get engine concurrent num
		engineCCNum := data["engineCCNum"].(int)
		if engineClusterEnabled {
			if engineCCNum%2 == 0 {
				engineCCNum = engineCCNum / engineClusterNodeNum
			} else {
				engineCCNum = engineCCNum/engineClusterNodeNum + 1
			}
		}
		// option setting
		op := Options{
			Dial:                 grpcDial,
			PoolModel:            vMap["POOL_MODEL"].(int),
			MaxIdle:              data["engineCCNum"].(int),
			MaxActive:            data["engineCCNum"].(int),
			MaxConcurrentStreams: 0,
			Reusable:             vMap["GRPC_REQUEST_REUSABLE"].(bool),
			RequestIdleTime:      vMap["REQUEST_IDLE_TIME"].(int),
			RequestMaxLife:       vMap["REQUEST_MAX_LIFE"].(int),
			RequestTimeOut:       vMap["REQUEST_TIMEOUT"].(int),
			GatewayProxyAddr:     serverAddr,
			GatewayProxyPort:     vMap["GATEWAY_PROXY_PORT"].(string),
			PoolStatus:           vMap["POOL_ENABLED"].(bool),
		}
		// create pool
		p, err := newGrpcPool(serverAddr, op)
		if err != nil {
			log.Fatalf("failed to new pool: %v", err)
		}
		// setting pool remote addr
		p.poolRemoteAddr = serverAddr
		p.name = data["sceneCode"].(string)
		// append pools
		if engineType == "asr" {
			asrOptions = op
			if NetWorkMode == STRICT_NETWORK_MODE {
				ASRGatewayProxyAddr = GatewayProxyAddr + ":" + asrOptions.GatewayProxyPort
			} else if NetWorkMode == GLOBAL_NETWORK_MODE {
				ASRGatewayProxyAddr = asrOptions.GatewayProxyPort
			}
			// asrPools = append(asrPools, p)
			asrPools[p.name] = p
		} else if engineType == "tts" {
			ttsOptions = op
			if NetWorkMode == STRICT_NETWORK_MODE {
				TTSGatewayProxyAddr = GatewayProxyAddr + ":" + ttsOptions.GatewayProxyPort
			} else if NetWorkMode == GLOBAL_NETWORK_MODE {
				TTSGatewayProxyAddr = ttsOptions.GatewayProxyPort
			}
			// ttsPools = append(ttsPools, p)
			ttsPools[p.name] = p
		}
	}
}

// init pool form tencent
func initPoolForTenant() {
	// multi tenant support
	if TenantEnabled {
		tenants = make(map[string]map[string]*Tenant)
		tenants = initTenantPool(map[string]interface{}{
			"asr": asrPools,
			"tts": ttsPools,
		})
	}
}

// init pool from exist engine
func initPoolFromExistEngine(data map[string]interface{}) {
	for {
		// get engine svcs
		engineType := data["engineType"].(string)
		logging.Log.Info("init " + engineType + " engine pool from exists engine ...")
		engineSvcs := kvs.GetEngineSvcs(map[string]string{
			data["engineSelectorKey"].(string): data["engineSelectorValue"].(string),
		})
		// range svcs
		for _, svc := range engineSvcs.Items {
			liveSvcStatus := checkGRPCSerer(svc.Name + ":" + engineSvcPort[engineType])
			if liveSvcStatus {
				data := strings.Split(svc.Name, "-")
				if len(data) == 3 {
					engineType := data[0]
					engineCCNum := data[1]
					sceneCode := data[2]
					engineName := sceneCode
					eCCNum, errECCNum := strconv.Atoi(engineCCNum)
					if errECCNum != nil {
						continue
					}
					// if engine exists
					switch engineType {
					case "asr":
						if _, ok := asrPools[engineName]; ok {
							if asrPools[engineName].capacity == int32(eCCNum) {
								// if eq continue or release pool
								continue
							} else {
								ReleaseGrpcPool(engineName, "asr")
							}

						}
					case "tts":
						if _, ok := ttsPools[engineName]; ok {
							if ttsPools[engineName].capacity == int32(eCCNum) {
								// if eq continue or release pool
								continue
							} else {
								ReleaseGrpcPool(engineName, "tts")
							}
						}
					}
					// get engine
					serverName := strings.ToLower(engineType) + "-" + engineCCNum + "-" + sceneCode
					ecNum, errEcNum := strconv.Atoi(engineCCNum)
					if errEcNum != nil {
						logging.Log.Error("init pool from exist engine strconv error:", errEcNum)
						return
					}

					podIp := ""
					endPoints, _ := kvs.GetEndpointsByName(serverName)
					for _, endPoint := range endPoints.Subsets {
						for _, ip := range endPoint.Addresses {
							podIp = ip.IP
						}
					}

					if podIp == "" {
						continue
					}

					// get engine pods
					// pods := kvs.GetEnginePods(map[string]string{
					// 	"engineName": engineName,
					// })

					// for _, pod := range pods.Items {
					// podReadyStats := false
					realServerHost := podIp + ":" + engineSvcPort[engineType]
					livePodStatus := checkGRPCSerer(realServerHost)
					// podStatus := string(pod.Status.Phase)
					// for _, condition := range pod.Status.Conditions {
					// 	if condition.Type == "Ready" {
					// 		if condition.Status == "True" && podStatus == "Running" {
					// 			podReadyStats = true
					// 		}
					// 	}
					// }
					if livePodStatus {
						// logging.Log.Info("discovered " + engineType + " engine pod (" + pod.Name + "), will init pool later ...")
						// init pool
						InitGrpcPoolByOmp(map[string]interface{}{
							"engineType":  engineType,
							"engineCCNum": ecNum,
							"sceneCode":   sceneCode,
							"serverHost":  podIp,
							"serverName":  serverName,
						})
						logging.Log.Info("init pool ", svc.Name, " finish ...")
					}
					// }
				}
			} else {
				logging.Log.Info("discovered " + engineType + " engine svc, but svc is unavailable ...")
			}
		}
		// setting sleep inverval time
		if time.Duration(enginePoolInitTime[engineType]) == 0 {
			time.Sleep(ENGINE_INIT_POOL_INTERVAL_TIME)
		} else {
			time.Sleep(time.Duration(enginePoolInitTime[engineType]) * time.Second)
		}
	}
}

// init pool from exist engine
func InitPoolFromExistEngineForUpdate(data map[string]interface{}) {
	// for {
	// get engine svcs
	engineType := data["engineType"].(string)
	logging.Log.Info("init " + engineType + " engine pool from exists engine ...")
	if engineType == "asr" {
		data["engineSelectorKey"] = engineSvcSelectorKey
		data["engineSelectorValue"] = asrEngineSvcSelectorVal
	} else if engineType == "tts" {
		data["engineSelectorKey"] = engineSvcSelectorKey
		data["engineSelectorValue"] = ttsEngineSvcSelectorVal
	}
	engineSvcs := kvs.GetEngineSvcs(map[string]string{
		data["engineSelectorKey"].(string): data["engineSelectorValue"].(string),
	})
	// range svcs
	for _, svc := range engineSvcs.Items {
		liveSvcStatus := checkGRPCSerer(svc.Name + ":" + engineSvcPort[engineType])
		if liveSvcStatus {
			data := strings.Split(svc.Name, "-")
			if len(data) == 3 {
				engineType := data[0]
				engineCCNum := data[1]
				sceneCode := data[2]
				engineName := sceneCode
				eCCNum, errECCNum := strconv.Atoi(engineCCNum)
				if errECCNum != nil {
					continue
				}
				// if engine exists
				switch engineType {
				case "asr":
					if _, ok := asrPools[engineName]; ok {
						if asrPools[engineName].capacity == int32(eCCNum) {
							// release pool
							ReleaseGrpcPool(engineName, "asr")
						}
					}
				case "tts":
					if _, ok := ttsPools[engineName]; ok {
						if ttsPools[engineName].capacity == int32(eCCNum) {
							// release pool
							ReleaseGrpcPool(engineName, "tts")
						}
					}
				}
				// get engine
				serverName := strings.ToLower(engineType) + "-" + engineCCNum + "-" + sceneCode
				ecNum, errEcNum := strconv.Atoi(engineCCNum)
				if errEcNum != nil {
					logging.Log.Error("init pool from exist engine strconv error:", errEcNum)
					return
				}
				// get engine pods
				// pods := kvs.GetEnginePods(map[string]string{
				// 	"engineName": engineName,
				// })

				podIp := ""
				endPoints, _ := kvs.GetEndpointsByName(serverName)
				for _, endPoint := range endPoints.Subsets {
					for _, ip := range endPoint.Addresses {
						podIp = ip.IP
					}
				}

				if podIp == "" {
					continue
				}

				realServerHost := podIp + ":" + engineSvcPort[engineType]
				livePodStatus := checkGRPCSerer(realServerHost)
				if livePodStatus {
					// logging.Log.Info("discovered " + engineType + " engine pod (" + pod.Name + "), will init pool later ...")
					// init pool
					InitGrpcPoolByOmp(map[string]interface{}{
						"engineType":  engineType,
						"engineCCNum": ecNum,
						"sceneCode":   sceneCode,
						"serverHost":  podIp,
						"serverName":  serverName,
					})
					logging.Log.Info("init pool ", svc.Name, " finish ...")
				}
			}
		} else {
			logging.Log.Info("discovered " + engineType + " engine svc, but svc is unavailable ...")
		}
	}
	// setting sleep inverval time
	if time.Duration(enginePoolInitTime[engineType]) == 0 {
		time.Sleep(ENGINE_INIT_POOL_INTERVAL_TIME)
	} else {
		time.Sleep(time.Duration(enginePoolInitTime[engineType]) * time.Second)
	}
	// }
}

// rpc server
func (plugin *Plugin) GRPCServer() {
	// init config
	logging.Log.Info("gRPC Server start ...")
	// get gRPC port
	defer func() {
		plugin.Status <- false
	}()
	lis, err := net.Listen("tcp", ":50002")
	if err != nil {
		logging.Log.Error("failed to listen: %v", err)
		return
	}
	// grpc new server
	srv := grpc.NewServer(grpc.CustomCodec(Codec()),
		grpc.UnknownServiceHandler(TransparentHandler(GrpcProxyTransport)))
	// register service
	RegisterService(srv, GrpcProxyTransport,
		"PingEmpty",
		"Ping",
		"PingError",
		"PingList",
	)
	reflection.Register(srv)
	// start ser listen
	err = srv.Serve(lis)
	if err != nil {
		logging.Log.Error("failed to serve: %v", err)
		return
	}
}

// new grpc pool
func newGrpcPool(address string, option Options) (*Pool, error) {
	dial := func() (*grpc.ClientConn, error) {
		return grpcDial(address)
	}

	// return Init(factor, init, size, 10*time.Second, 60*time.Second, 10*time.Second, STRICT_MODE)
	gp, err := NewPool(
		dial,
		int32(option.MaxIdle),
		int32(option.MaxActive),
		time.Duration(option.RequestIdleTime)*time.Second,
		time.Duration(option.RequestMaxLife)*time.Second,
		time.Duration(option.RequestTimeOut)*time.Second,
		option.PoolModel,
	)
	return gp, err
}

// release grpc pool
func ReleaseGrpcPool(poolName, engineType string) {
	switch strings.ToLower(engineType) {
	case "asr":
		if _, ok := asrPools[poolName]; ok {
			asrPools[poolName].Close()
			delete(asrPools, poolName)
		}
	case "tts":
		if _, ok := ttsPools[poolName]; ok {
			ttsPools[poolName].Close()
			delete(ttsPools, poolName)
		}
	}

	logging.Log.Info("delete ", engineType, " pool ", poolName, " success")
}

// grpc dial
func grpcDial(address string) (*grpc.ClientConn, error) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), DialTimeout)
	defer ctxCancel()
	gcc, err := grpc.DialContext(ctx, address,
		grpc.WithCodec(Codec()),
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(BackoffMaxDelay),
		grpc.WithInitialWindowSize(InitialWindowSize),
		grpc.WithInitialConnWindowSize(InitialConnWindowSize),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxSendMsgSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxRecvMsgSize)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                KeepAliveTime,
			Timeout:             KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
	)

	if err != nil {
		logging.Log.Error("grpc dial failed !", err)
	}

	return gcc, err
}

// check grpc server
func checkGRPCSerer(addr string) bool {
	// 3 秒超时
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil || conn == nil {
		return false
	}
	defer conn.Close()
	return true
}

func CheckEngineSererStatus(data map[string]interface{}) bool {
	// 3 秒超时
	addr := data["serverName"].(string) + ":" + engineSvcPort[data["engineType"].(string)]
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil || conn == nil {
		return false
	}
	defer conn.Close()
	return true
}

// check asr grpc server task
func checkAsrGRPCSererHealthTask() {
	for {
		// asr pool check
		for _, pool := range asrPools {
			checkStatus := checkGRPCSerer(pool.poolRemoteAddr)
			if !checkStatus {
				for i := 0; i < 5; i++ {
					time.Sleep(1 * time.Second)
					checkStatus = checkGRPCSerer(pool.poolRemoteAddr)
					if !checkStatus {
						continue
					} else {
						break
					}
				}
			}
			// check status
			if !checkStatus {
				logging.Log.Error("gRPC Server is down !")
				pool.status = false
				// exist failed pool
				failedPoolStatus = true
			} else {
				// logging.Log.Info("gRPC Server is up !")
				pool.status = true
				// failed pool is up
				failedPoolStatus = false
			}
		}
		time.Sleep(time.Second)
	} // end for
}

// check tts grpc server task
func checkTtsGRPCSererHealthTask() {
	for {
		// tts pool check
		for _, pool := range ttsPools {
			checkStatus := checkGRPCSerer(pool.poolRemoteAddr)
			if !checkStatus {
				for i := 0; i < 5; i++ {
					time.Sleep(1 * time.Second)
					checkStatus = checkGRPCSerer(pool.poolRemoteAddr)
					if !checkStatus {
						continue
					} else {
						break
					}
				}
			}
			// check status
			if !checkStatus {
				logging.Log.Error("gRPC Server is down !")
				pool.status = false
				// exist failed pool
				failedPoolStatus = true
			} else {
				// logging.Log.Info("gRPC Server is up !")
				pool.status = true
				// failed pool is up
				failedPoolStatus = false
			}
		}
		time.Sleep(time.Second)
	} // end for
}

// get pool metrics data
func GetAsrPoolMetricsData() (map[string]interface{}, bool) {
	metricsDataMap := make(map[string]interface{})
	mDataMap := make([]map[string]interface{}, 0)
	// pool status
	if !asrOptions.PoolStatus {
		mDataMap = append(mDataMap, map[string]interface{}{
			"engineName":      "",
			"connCurrent":     0,
			"poolCurrentSize": 0,
			"poolSize":        0,
			"poolRemoteAddr":  "",
		})
		metricsDataMap["engines"] = mDataMap
		return metricsDataMap, asrOptions.PoolStatus
	}

	// engine data
	for _, asrPool := range asrPools {
		if !asrPool.status {
			continue
		}

		connCurrent := asrPool.GetConnCurrent()
		poolCurrentSize := asrPool.Size()
		if poolCurrentSize >= int(asrPool.capacity) {
			connCurrent = 0
		}
		// data map
		mDataMap = append(mDataMap, map[string]interface{}{
			"engineName":      asrPool.name,
			"connCurrent":     connCurrent,
			"poolCurrentSize": poolCurrentSize,
			"poolSize":        int(asrPool.capacity),
			"poolRemoteAddr":  asrPool.poolRemoteAddr,
		})
	}
	metricsDataMap["engines"] = mDataMap

	// tenant data
	if TenantEnabled {
		mTenantDataMap := make([]map[string]interface{}, 0)
		for _, tenantPool := range tenants["asr"] {
			connCurrent := tenantPool.GetConnCurrent()
			poolCurrentSize := tenantPool.Size()
			if poolCurrentSize >= int(tenantPool.capacity) {
				connCurrent = 0
			}
			mTenantDataMap = append(mTenantDataMap, map[string]interface{}{
				"tenantId":        tenantPool.id.(string),
				"tenantName":      tenantPool.tenantName,
				"connCurrent":     connCurrent,
				"poolCurrentSize": poolCurrentSize,
				"poolSize":        int(tenantPool.capacity),
				"poolRemoteAddr":  tenantPool.poolRemoteAddr,
			})
		}
		metricsDataMap["tenants"] = mTenantDataMap
	}
	logging.Log.Info("-------------------- engine metrics data, asr size ", len(asrPools))
	// return data
	return metricsDataMap, asrOptions.PoolStatus
}

// get pool metrics data
func GetTtsPoolMetricsData() (map[string]interface{}, bool) {
	metricsDataMap := make(map[string]interface{})
	mDataMap := make([]map[string]interface{}, 0)
	// pool status
	if !ttsOptions.PoolStatus {
		mDataMap = append(mDataMap, map[string]interface{}{
			"engineName":      "",
			"connCurrent":     0,
			"poolCurrentSize": 0,
			"poolSize":        0,
			"poolRemoteAddr":  "",
		})
		metricsDataMap["engines"] = mDataMap
		return metricsDataMap, ttsOptions.PoolStatus
	}

	// engine data
	for _, ttsPool := range ttsPools {
		if !ttsPool.status {
			continue
		}

		connCurrent := ttsPool.GetConnCurrent()
		poolCurrentSize := ttsPool.Size()
		if poolCurrentSize >= int(ttsPool.capacity) {
			connCurrent = 0
		}
		// data map
		mDataMap = append(mDataMap, map[string]interface{}{
			"engineName":      ttsPool.name,
			"connCurrent":     int(connCurrent),
			"poolCurrentSize": poolCurrentSize,
			"poolSize":        int(ttsPool.capacity),
			"poolRemoteAddr":  ttsPool.poolRemoteAddr,
		})
	}
	metricsDataMap["engines"] = mDataMap

	// tenant data
	if TenantEnabled {
		mTenantDataMap := make([]map[string]interface{}, 0)
		for _, tenantPool := range tenants["tts"] {
			connCurrent := tenantPool.GetConnCurrent()
			poolCurrentSize := tenantPool.Size()
			if poolCurrentSize >= int(tenantPool.capacity) {
				connCurrent = 0
			}
			mTenantDataMap = append(mTenantDataMap, map[string]interface{}{
				"tenantId":        tenantPool.id.(string),
				"tenantName":      tenantPool.tenantName,
				"connCurrent":     connCurrent,
				"poolCurrentSize": poolCurrentSize,
				"poolSize":        int(tenantPool.capacity),
				"poolRemoteAddr":  tenantPool.poolRemoteAddr,
			})
		}

		metricsDataMap["tenants"] = mTenantDataMap
	}

	// return data
	logging.Log.Info("-------------------- engine metrics data, tts size ", len(ttsPools))
	return metricsDataMap, ttsOptions.PoolStatus
}

// encode (support base64)
func strEncode(str []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(str))
}

// decode (support base64)
func strDecode(str []byte) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(str))
}
