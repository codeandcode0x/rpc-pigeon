package service

import (
	"errors"
	"rpc-gateway/pkg/plugins/httpserver/model"
	"rpc-gateway/pkg/plugins/pool/grpc"
	"rpc-gateway/pkg/plugins/vs/kvs"
)

type VSService struct {
	DAO model.MetricsDAO
}

// get user service
func (s *VSService) getSvc() *VSService {
	var m model.BaseModel
	return &VSService{
		DAO: &model.MetricsData{BaseModel: m},
	}
}

// create engine
func (s *VSService) CreateEngine(data map[string]interface{}) error {
	return kvs.CreateEngine(data)
}

// update engine
func (s *VSService) UpdateEngine(data map[string]interface{}) error {
	return kvs.UpdateEngine(data)
}

// delete engine
func (s *VSService) DeleteEngine(data map[string]interface{}) error {
	return kvs.DeleteEngine(data)
}

// check engine status
func (s *VSService) CheckEngine(data map[string]interface{}) error {
	errCheckEngine := kvs.CheckEngine(map[string]interface{}{
		"engineName":  data["engineName"],
		"engineCCNum": data["engineCCNum"],
	})
	if errCheckEngine != nil {
		return errCheckEngine
	}
	// check engine status
	if !grpc.CheckEngineSererStatus(data) {
		return errors.New("engine create succeed, but engine svc is unavailable")
	}

	return nil
}
