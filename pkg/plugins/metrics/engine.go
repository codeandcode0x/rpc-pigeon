package metrics

import (
	"encoding/json"
	logging "rpc-gateway/pkg/core/log"
	"rpc-gateway/pkg/plugins"
	"rpc-gateway/pkg/plugins/httpserver/util"
	"rpc-gateway/pkg/plugins/pool/grpc"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Metricser interface {
	ASRMetrics()
	TTSMetrics()
}

type MetricsData struct {
	MetricsName   string `json:"metricsName"`
	MetricsData   string `json:"metricsData"`
	MetricsModule string `json:"metricsModule"`
	MetricsStatus bool   `json:"metricsStatus"`
}

type Plugin plugins.Plugin

// asr metrics
func ASRMetrics() []MetricsData {
	mDatas := []MetricsData{}
	// get ASR metrics data
	mDataMap, poolStatus := grpc.GetAsrPoolMetricsData()
	tenants := mDataMap["tenants"]
	engines := mDataMap["engines"]

	for _, mData := range engines.([]map[string]interface{}) {
		// json data
		dataBytes, _ := json.Marshal(map[string]interface{}{
			"engineName":      mData["engineName"],
			"connCurrent":     mData["connCurrent"],
			"poolCurrentSize": mData["poolCurrentSize"],
			"poolSize":        mData["poolSize"],
			"poolRemoteAddr":  mData["poolRemoteAddr"],
		})
		mDatas = append(mDatas, MetricsData{
			MetricsName:   "engineData",
			MetricsModule: "ASR",
			MetricsStatus: poolStatus,
			MetricsData:   string(dataBytes),
		})
	}

	if tenants != nil {
		for _, mData := range tenants.([]map[string]interface{}) {
			dataBytes, _ := json.Marshal(map[string]interface{}{
				"tenantId":        mData["tenantId"],
				"tenantName":      mData["tenantName"],
				"connCurrent":     mData["connCurrent"],
				"poolCurrentSize": mData["poolCurrentSize"],
				"poolSize":        mData["poolSize"],
				"poolRemoteAddr":  mData["poolRemoteAddr"],
			})
			mDatas = append(mDatas, MetricsData{
				MetricsName:   "tenantEngineData",
				MetricsModule: "ASR",
				MetricsStatus: poolStatus,
				MetricsData:   string(dataBytes),
			})
		}
	}

	return mDatas
}

// tts metrics
func TTSMetrics() []MetricsData {
	mDatas := []MetricsData{}
	// get ASR metrics data
	mDataMap, poolStatus := grpc.GetTtsPoolMetricsData()
	tenants := mDataMap["tenants"]
	engines := mDataMap["engines"]

	for _, mData := range engines.([]map[string]interface{}) {
		// json data
		dataBytes, _ := json.Marshal(map[string]interface{}{
			"engineName":      mData["engineName"],
			"connCurrent":     mData["connCurrent"],
			"poolCurrentSize": mData["poolCurrentSize"],
			"poolSize":        mData["poolSize"],
			"poolRemoteAddr":  mData["poolRemoteAddr"],
		})
		mDatas = append(mDatas, MetricsData{
			MetricsName:   "engineData",
			MetricsModule: "TTS",
			MetricsStatus: poolStatus,
			MetricsData:   string(dataBytes),
		})
	}

	if tenants != nil {
		for _, mData := range tenants.([]map[string]interface{}) {
			dataBytes, _ := json.Marshal(map[string]interface{}{
				"tenantId":        mData["tenantId"],
				"tenantName":      mData["tenantName"],
				"connCurrent":     mData["connCurrent"],
				"poolCurrentSize": mData["poolCurrentSize"],
				"poolSize":        mData["poolSize"],
				"poolRemoteAddr":  mData["poolRemoteAddr"],
			})
			mDatas = append(mDatas, MetricsData{
				MetricsName:   "tenantEngineData",
				MetricsModule: "TTS",
				MetricsStatus: poolStatus,
				MetricsData:   string(dataBytes),
			})
		}
	}

	return mDatas
}

// metrics server
func (plugin *Plugin) ShowMetrics() {
	// get gRPC port
	defer func() {
		plugin.Status <- false
	}()

	util.InitConfig()

	if strings.ToLower(viper.GetString("RUN_MODE")) == "dev" {
		for {
			logging.Log.Info(ASRMetrics(), TTSMetrics())
			time.Sleep(1 * time.Second)
		}
	}
}
