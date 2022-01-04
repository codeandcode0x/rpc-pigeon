package controller

import (
	"net/http"
	"rpc-gateway/pkg/plugins/httpserver/service"
	"rpc-gateway/pkg/plugins/httpserver/util"

	"github.com/gin-gonic/gin"
)

// controller struct
type MetricsController struct {
	apiVersion string
	Service    *service.MetricsService
}

// get controller
func (uc *MetricsController) getCtl() *MetricsController {
	var svc *service.MetricsService
	return &MetricsController{"v1", svc}
}

// User godoc
// @Summary calc User api
// @Description get all users
// @Tags users
// @Accept json
// @Produce json
// @Param apiQuery query string false "used for api"
// @Success 200 {integer} string "{"code":0, "data":[{User},{User}], "message":"ok"}"
// @Failure 404 {string} string "method not found"
// @Failure 500 {string} string "Service Unavailable"
// @Router /users [get]
func (uc *MetricsController) GetEngineMetricsData(c *gin.Context) {

	mDatas, err := uc.getCtl().Service.GetMetricsData()
	// error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": err,
		})
		return
	}
	// no data
	if len(mDatas) < 1 {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "no data",
		})
		return
	}
	// send message
	util.SendMessage(c, util.Message{
		Code:    0,
		Message: "OK",
		Data:    mDatas,
	})
}
