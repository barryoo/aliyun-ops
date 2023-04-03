package main

import (
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	gr "github.com/awesome-fc/golang-runtime"
	"github.com/joho/godotenv"
	"os"
)

func initialize(ctx *gr.FCContext) error {
	ctx.GetLogger().Infoln("init golang!")
	return nil
}

func handler(ctx *gr.FCContext, eventByte []byte) ([]byte, error) {
	fcLogger := ctx.GetLogger()
	_, err := json.Marshal(ctx)
	if err != nil {
		fcLogger.Error("error:", err)
	}
	fcLogger.Infof("hello golang!")
	fcLogger.Infof("event: %s", string(eventByte))

	//对eventBody解析, 得到实例的常用信息
	var event event
	err = json.Unmarshal(eventByte, &event)
	if err != nil {
		return nil, err
	}
	fcLogger.Infof("event: %s", event)

	fcLogger.Infof("Access: %s", os.Getenv("Access"))
	fcLogger.Infof("RegionId: %s", os.Getenv("RegionId"))
	fcLogger.Infof("access: %s", os.Getenv("access"))
	fcLogger.Infof("region: %s", os.Getenv("region"))

	//如果event.EventType为"StatusNotification", 并且event.name="Instance:PreemptibleInstanceInterruption", 则继续, 否则返回
	if event.EventType != "StatusNotification" || event.Name != "Instance:PreemptibleInstanceInterruption" {
		return eventByte, nil
	}

	//实例标识
	regionId := event.RegionID
	instanceId := event.Content.InstanceID
	instanceName := event.InstanceName
	fcLogger.Infof(" resionId: %s, instanceId: %s, instanceName: %s", regionId, instanceId, instanceName)

	//ecs client
	clients := Clients{}
	ecsCli := clients.GetECSClient()
	//查询ecs实例的信息
	instancesRequest := ecs.CreateDescribeInstancesRequest()
	instancesRequest.RegionId = regionId
	instanceIdByte, err := json.Marshal([]string{instanceId})
	if err != nil {
		return nil, err
	}
	instancesRequest.InstanceIds = string(instanceIdByte)
	res, err := ecsCli.DescribeInstances(instancesRequest)
	if res.Instances.Instance[0].OperationLocks.LockReason != nil {
		//LockReason: financial：因欠费被锁定。 security：因安全原因被锁定。Recycling：抢占式实例的待释放锁定状态。 dedicatedhostfinancial：因为专有宿主机欠费导致ECS实例被锁定。 refunded：因退款被锁定。
		//打印所有的锁定原因
		for _, lockReason := range res.Instances.Instance[0].OperationLocks.LockReason {
			fcLogger.Infof("lockReason: %s", lockReason)
		}
	}

	//查询ecs实例的状态信息
	instanceStatusRequest := ecs.CreateDescribeInstanceStatusRequest()
	instanceStatusRequest.RegionId = event.RegionID
	instanceStatusRequest.InstanceId = &([]string{event.Content.InstanceID})
	instanceStatusRes, err := ecsCli.DescribeInstanceStatus(instanceStatusRequest)
	if err != nil {
		return nil, err
	}
	//打印所有的实例状态
	for _, instanceStatus := range instanceStatusRes.InstanceStatuses.InstanceStatus {
		fcLogger.Infof("instanceStatus: %s", instanceStatus.Status)
	}

	//
	return eventByte, nil
}

func main() {
	gr.Start(handler, initialize)
}

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
}
