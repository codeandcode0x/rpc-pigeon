package grpc

import (
	"context"
	"math/rand"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// options struct
type Options struct {
	Dial                 func(address string) (*grpc.ClientConn, error)
	PoolModel            int    // Pool 模型
	MaxIdle              int    // 最大空闲数量
	MaxActive            int    // 最大活跃连接数
	MaxConcurrentStreams int    // 最大并发数量
	Reusable             bool   // 是否可以复用
	RequestIdleTime      int    // request idle 时间
	RequestMaxLife       int    // request max life 时间
	RequestTimeOut       int    // request timeout
	GatewayProxyAddr     string // grpc gateway 代理地址
	GatewayProxyPort     string // grpc gateway 代理端口
	PoolStatus           bool   // 连接池是否开启
}

// grpc proxy director
func GrpcProxyTransport(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, *Client, error) {
	if strings.HasPrefix(fullMethodName, "/ivc.v1.internal") {
		return nil, nil, nil, status.Errorf(codes.Unimplemented, "invaild or unsupported method")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	outCtx, _ := context.WithCancel(ctx)
	outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())
	var err error
	var conn *Client

	if ok {
		serverAddr := ""
		if addrMdData, exists := md[":authority"]; exists {
			serverAddr = addrMdData[0]
			if NetWorkMode == GLOBAL_NETWORK_MODE {
				addrPort := strings.Split(addrMdData[0], ":")
				serverAddr = addrPort[1]
			}
		}
		// token code : userid + sceneCode + rand
		if serverAddr == ASRGatewayProxyAddr {
			if OMPEnabled || TenantEnabled {
				// get token
				if token, tokenExists := md["token"]; tokenExists {
					engineCodes, errDecode := strDecode([]byte(token[0]))
					if errDecode != nil {
						status.Errorf(codes.Unimplemented, "token decode failed .")
					}
					// get engine code
					engineCode := strings.Split(string(engineCodes), "-")
					if len(engineCode) < 2 {
						return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown engine token")
					}
					// enabled omp
					if OMPEnabled {
						if _, ok := asrPools[engineCode[1]]; !ok {
							return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown engine pool")
						}
						conn, err = asrPools[engineCode[1]].Acquire(ctx)
					} else if TenantEnabled { // enabled tenant
						if _, ok := tenants["asr"][engineCode[1]]; !ok {
							return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown engine pool")
						}
						conn, err = tenants["asr"][engineCode[1]].Acquire(ctx)
					}
				} else {
					conn, err = nil, status.Errorf(codes.Unimplemented, "unknown engine pool")
				} // end token exist
			} else { // don't support omp and tenant
				conn, err = balancePool(asrPools).Acquire(ctx)
			}
		} else if serverAddr == TTSGatewayProxyAddr {
			if OMPEnabled || TenantEnabled {
				// get token
				if token, tokenExists := md["token"]; tokenExists {
					engineCodes, errDecode := strDecode([]byte(token[0]))
					if errDecode != nil {
						status.Errorf(codes.Unimplemented, "token decode failed .")
					}
					// get engine code
					engineCode := strings.Split(string(engineCodes), "-")
					if len(engineCode) < 2 {
						return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown engine token")
					}
					// enabled omp
					if OMPEnabled { // enabled omp
						if _, ok := ttsPools[engineCode[1]]; !ok {
							return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown engine pool")
						}
						conn, err = ttsPools[engineCode[1]].Acquire(ctx)
					} else if TenantEnabled { // enabled tenant
						if _, ok := tenants["tts"][engineCode[1]]; !ok {
							return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown engine pool")
						}
						conn, err = tenants["tts"][engineCode[1]].Acquire(ctx)
					}
				} else { // end token exist
					conn, err = nil, status.Errorf(codes.Unimplemented, "unknown engine pool")
				}
			} else { // don't support omp and tenant
				conn, err = balancePool(ttsPools).Acquire(ctx)
			}
		}

		if conn != nil {
			return outCtx, conn.ClientConn, conn, err
		}
	}
	return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown method")
}

// 负载均衡
func balancePool(pools map[string]*Pool) *Pool {
	var sumSize, size int
	var index string
	// if len(pools) == 1 {
	// 	return pools[index]
	// }
	if len(pools) < 1 {
		return nil
	}

	if len(pools) == 1 {
		for k, _ := range pools {
			index = k
		}
	} else {
		// 判断是否存在失效的连接池
		// if failedPoolStatus {
		var indexRand []string
		// 最小连接数
		for k, pool := range pools {
			if !pool.status {
				continue
			}

			pSize := pool.Size()
			sumSize += int(pool.Size())
			if size <= pSize {
				size = int(pSize)
				index = k
			}
			indexRand = append(indexRand, k)
		}
		// 随机
		if sumSize == 0 {
			index = indexRand[rand.Intn(len(indexRand))]
		}
		// } else {
		// 	// 最小连接数
		// 	for k, pool := range pools {
		// 		if !pool.status {
		// 			continue
		// 		}
		// 		pSize := pool.Size()
		// 		sumSize += int(pool.Size())
		// 		if size < pSize {
		// 			size = int(pSize)
		// 			index = k
		// 		}
		// 	}
		// 	// 随机
		// 	if sumSize == 0 {
		// 		index = rand.Intn(len(pools))
		// 	}
		// }
	}

	return pools[index]
}

// 负载均衡
func balanceTenantPool(pools []*Tenant) *Tenant {
	var index, sumSize, size int
	if len(pools) == 1 {
		return pools[index]
	}
	// 最小连接数
	for k, pool := range pools {
		pSize := pool.Size()
		sumSize += int(pool.Size())
		if size < pSize {
			size = int(pSize)
			index = k
		}
	}
	// 随机
	if sumSize == 0 {
		index = rand.Intn(len(pools))
	}
	return pools[index]
}
