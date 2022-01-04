package service

import (
	"rpc-gateway/pkg/plugins/httpserver/model"
	"rpc-gateway/pkg/plugins/metrics"
	"time"
)

type MetricsService struct {
	DAO model.MetricsDAO
}

// get user service
func (s *MetricsService) getSvc() *MetricsService {
	var m model.BaseModel
	return &MetricsService{
		DAO: &model.MetricsData{BaseModel: m},
	}
}

func (s *MetricsService) GetMetricsData() ([]model.MetricsData, error) {
	var mDatas []model.MetricsData
	var datas []metrics.MetricsData

	datas = metrics.ASRMetrics()
	for _, data := range datas {
		if data.MetricsStatus {
			mDatas = append(mDatas, model.MetricsData{
				MetricsName:   data.MetricsName,
				MetricsData:   data.MetricsData,
				MetricsModule: data.MetricsModule,
				BaseModel: model.BaseModel{
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				},
			})
		}
	}

	datas = metrics.TTSMetrics()
	for _, data := range datas {
		if data.MetricsStatus {
			mDatas = append(mDatas, model.MetricsData{
				MetricsName:   data.MetricsName,
				MetricsData:   data.MetricsData,
				MetricsModule: data.MetricsModule,
				BaseModel: model.BaseModel{
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				},
			})
		}
	}

	return mDatas, nil
}
