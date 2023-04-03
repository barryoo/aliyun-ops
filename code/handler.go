package main

import (
	"encoding/json"
	"os"

	aliyunEcs "businessmatics.io/aliyun-ops/aliyun/ecs"
	"businessmatics.io/aliyun-ops/utils"
	gr "github.com/awesome-fc/golang-runtime"
)

func handler(ctx *gr.FCContext, eventByte []byte) ([]byte, error) {
	fcLogger := ctx.GetLogger()

	defer func() {
		if r := recover(); r != nil {
			fcLogger.Error("recover:", r)
			return
		}
	}()

	_, err := json.Marshal(ctx)
	utils.E("Marshal FCContext", err)
	fcLogger.Infof("event: %s", string(eventByte))

	//对eventBod解析, 得到实例的常用信息
	var event aliyunEcs.Event
	err = json.Unmarshal(eventByte, &event)
	utils.E("Unmarshal event", err)

	fcLogger.Infof("Access: %s", os.Getenv("Access"))
	fcLogger.Infof("RegionId: %s", os.Getenv("RegionId"))
	fcLogger.Infof("access: %s", os.Getenv("access"))
	fcLogger.Infof("region: %s", os.Getenv("region"))

	//如果event.EventType为"StatusNotification", 并且event.name="Instance:PreemptibleInstanceInterruption", 则继续, 否则返回
	// if event.EventType != "StatusNotification" && event.Name != "Instance:PreemptibleInstanceInterruption" {
	// 	fcLogger.Info("这不是抢占式实例终端事件, 不处理")
	// 	return nil, nil
	// }

	if event.EventType == "StatusNotification" && event.Name == "Instance:PreemptibleInstanceInterruption" {
		processor := aliyunEcs.NewProcessor(ctx, event, os.Getenv("RegionId"), event.Content.InstanceID, event.InstanceName)
		return processor.Process()
	}
	return nil, nil

}
