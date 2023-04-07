package main

import (
	aliyunEcs "businessmatics.io/aliyun-ops/aliyun/ecs"
	"businessmatics.io/aliyun-ops/asset"
	"businessmatics.io/aliyun-ops/utils"
	"encoding/json"
	gr "github.com/awesome-fc/golang-runtime"
	"github.com/driftprogramming/godotenv"
	"log"
	"os"
)

func initialize(ctx *gr.FCContext) error {
	ctx.GetLogger().Infoln("init golang!")
	return nil
}

func main() {
	gr.Start(handler, initialize)
}

func init() {
	err := godotenv.Load(asset.Env, ".env")
	if err != nil {
		log.Printf("Error loading .env file, err: %s", err)
		panic(err)
	}
}

func handler(ctx *gr.FCContext, eventByte []byte) ([]byte, error) {
	fcLogger := ctx.GetLogger()

	defer func() {
		if r := recover(); r != nil {
			fcLogger.Error("recover:", r)
			return
		}
	}()

	_, err := json.Marshal(ctx)
	utils.P("Marshal FCContext", err)
	fcLogger.Infof("event: %s", string(eventByte))

	//对eventBod解析, 得到实例的常用信息
	var event aliyunEcs.Event
	err = json.Unmarshal(eventByte, &event)
	utils.P("Unmarshal event", err)

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
		processor := aliyunEcs.NewProcessor(ctx, event, os.Getenv("RegionId"), event.Content.InstanceID)
		return processor.Process()
	}
	return nil, nil

}
