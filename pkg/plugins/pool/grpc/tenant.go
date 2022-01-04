package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

type Tenant struct {
	id             interface{}
	clients        chan *Client
	connCurrent    int32        // 当前连接数
	capacity       int32        // 容量
	size           int32        // 容量大小 (动态变化)
	lock           sync.RWMutex // 读写锁
	mode           int          // 连接池 模型
	poolRemoteAddr string       // 远程连接地址
	tenantName     string       // tenant 名称
	engine         string       // 引擎名称
}

var tenantViper *viper.Viper

// acquire tenant pool
func initTenantPool(pools map[string]interface{}) map[string]map[string]*Tenant {
	tenantViper = viper.New()
	tenantViper.SetConfigName("TenantConfig")
	tenantViper.SetConfigType("yaml")
	tenantViper.AddConfigPath("config/")
	err := tenantViper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error get tenant config file: %s", err))
	}

	// acquire pools
	tenants := make(map[string]map[string]*Tenant)
	tenants["asr"] = make(map[string]*Tenant)
	tenants["tts"] = make(map[string]*Tenant)
	tvRoot := tenantViper.AllSettings()["tenants"]
	for _, v := range tvRoot.([]interface{}) {
		vMap := v.(map[interface{}]interface{})
		tenantPoolSize := vMap["ENGINE_POOL_SIZE"].(int)
		engine := strings.ToLower(vMap["ENGINE_NAME"].(string))
		tenantId := vMap["SCENE_CODE"].(string)

		tenant := &Tenant{
			id:             tenantId,
			clients:        make(chan *Client, tenantPoolSize),
			connCurrent:    0,
			capacity:       int32(tenantPoolSize),
			size:           int32(tenantPoolSize),
			lock:           sync.RWMutex{},
			mode:           0,
			poolRemoteAddr: "",
			tenantName:     vMap["TENANT_NAME"].(string),
			engine:         engine,
		}

		switch engine {
		case "asr":
			asrPools := pools["asr"].(map[string]*Pool)
			for i := 0; i < tenantPoolSize; i++ {
				client, err := blanceAcquirePool(asrPools)
				if err != nil {
					panic("asr pool capacity is not enough")
				}
				client.tenant = tenant
				tenant.clients <- client
			}

			tenants["asr"][tenantId] = tenant
		case "tts":
			ttsPools := pools["tts"].(map[string]*Pool)
			for i := 0; i < tenantPoolSize; i++ {
				client, err := blanceAcquirePool(ttsPools)
				if err != nil {
					panic("tts pool capacity is not enough")
				}
				client.tenant = tenant
				tenant.clients <- client
			}
			tenants["tts"][tenantId] = tenant
		}
	}

	return tenants
}

// blance acqurire chan
func blanceAcquirePool(pools map[string]*Pool) (*Client, error) {
	var sumSize, size int
	var index string
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
		// index = rand.Intn(len(pools))
		return nil, errors.New(" pool is empty")
	}

	return <-pools[index].clients, nil
}

// 从连接池取出一个连接
func (pool *Tenant) Acquire(ctx context.Context) (*Client, error) {
	if pool.IsClose() {
		return nil, errors.New("Pool is closed")
	}

	return <-pool.clients, nil
}

// 连接池中连接数
func (pool *Tenant) Size() int {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return len(pool.clients)
}

// 连接池是否关闭
func (pool *Tenant) IsClose() bool {
	return pool == nil || pool.clients == nil
}

// 实际连接数
func (pool *Tenant) GetConnCurrent() int32 {
	return pool.capacity - int32(pool.Size())
}
