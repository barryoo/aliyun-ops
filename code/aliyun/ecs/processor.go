package ecs

import (
	"encoding/json"
	"fmt"
	"time"

	"businessmatics.io/aliyun-ops/aliyun"
	"businessmatics.io/aliyun-ops/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	gr "github.com/awesome-fc/golang-runtime"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	// contains filtered or unexported fields
	ctx          *gr.FCContext
	fcLogger     *logrus.Entry
	event        Event
	regionId     string
	instanceId   string
	instanceName string
	clients      aliyun.Clients
}

func NewProcessor(ctx *gr.FCContext, event Event, regionId string, instanceId string, instanceName string) *Processor {
	return &Processor{
		ctx:          ctx,
		event:        event,
		regionId:     regionId,
		instanceId:   instanceId,
		instanceName: instanceName,
		fcLogger:     ctx.GetLogger(),
		clients:      aliyun.Clients{},
	}
}

func (p *Processor) Process() ([]byte, error) {
	p.fcLogger.Infof(" resionId: %s, instanceId: %s, instanceName: %s", p.regionId, p.instanceId, p.instanceName)
	instance := p.describeInstance()
	p.describeInstanceStatus()
	imageId := p.createImage()
	p.fcLogger.Infof("imageId: %s", imageId)
	p.describeImageStatus(imageId)
	//创建新的抢占式实例
	//1. 优先创建与原实例规格一致的实例
	//2. 查询可用区的实例规格, 如果没有与原实例规格一致的, 则创建与原实例规格最接近的实例
	zoneId := instance.ZoneId
	instanceType := instance.InstanceType
	//查询可用区的实例规格
	instanceTypes := p.describeInstanceTypes(zoneId)
	//查询可用区的实例规格
	instanceType = p.findInstanceType(instanceTypes, instanceType)
	p.fcLogger.Infof("instanceType: %s", instanceType)
	//创建新的抢占式实例
	instanceId := p.createInstance(zoneId, instanceType, imageId)
	p.fcLogger.Infof("instanceId: %s", instanceId)
	p.describeInstanceStatus()

	//释放旧的抢占式实例

	return []byte("success"), nil
}

//查询即将被释放的实例的信息
func (p *Processor) describeInstance() ecs.Instance {
	instancesRequest := ecs.CreateDescribeInstancesRequest()
	instancesRequest.RegionId = p.regionId
	instanceIdByte, err := json.Marshal([]string{p.instanceId})
	utils.E("Marshal DescribeInstancesRequest.InstanceIds", err)
	instancesRequest.InstanceIds = string(instanceIdByte)
	res, err := p.clients.GetECSClient().DescribeInstances(instancesRequest)
	utils.E("DescribeInstances", err)
	if res.Instances.Instance[0].OperationLocks.LockReason != nil {
		// 打印所有的锁定原因 LockReason: financial：因欠费被锁定。 security：因安全原因被锁定。Recycling：抢占式实例的待释放锁定状态。 dedicatedhostfinancial：因为专有宿主机欠费导致ECS实例被锁定。 refunded：因退款被锁定。
		for _, lockReason := range res.Instances.Instance[0].OperationLocks.LockReason {
			p.fcLogger.Infof("lockReason: %s", lockReason)
		}
	}
	return res.Instances.Instance[0]
}

//查询即将被释放的实例的状态
func (p *Processor) describeInstanceStatus() ecs.InstanceStatuses {
	instanceStatusRequest := ecs.CreateDescribeInstanceStatusRequest()
	instanceStatusRequest.RegionId = p.regionId
	instanceStatusRequest.InstanceId = &([]string{p.instanceId})
	ecsCli := p.clients.GetECSClient()
	instanceStatusRes, err := ecsCli.DescribeInstanceStatus(instanceStatusRequest)
	utils.E("DescribeInstanceStatus", err)
	instanceStatuses := instanceStatusRes.InstanceStatuses
	//打印所有的实例状态.
	for _, instanceStatus := range instanceStatusRes.InstanceStatuses.InstanceStatus {
		p.fcLogger.Infof("instanceStatus: %s", instanceStatus)
	}
	return instanceStatuses
}

//基于即将被释放的实例,创建镜像
func (p *Processor) createImage() string {
	createImageRequest := ecs.CreateCreateImageRequest()
	createImageRequest.RegionId = p.regionId
	createImageRequest.InstanceId = p.instanceId
	createImageRequest.ImageName = p.instanceName
	ecsCli := p.clients.GetECSClient()
	createImageRes, err := ecsCli.CreateImage(createImageRequest)
	utils.E("CreateImage", err)
	return createImageRes.ImageId
}

//查询镜像的状态, 循环查询,直到镜像状态为Available, 则表示镜像创建成功, 否则等待5秒,继续查询镜像状态. 最多重试12次
func (p *Processor) describeImageStatus(imageId string) (image ecs.Image, err error) {
	for i := 0; i < 12; i++ {
		describeImagesRequest := ecs.CreateDescribeImagesRequest()
		describeImagesRequest.RegionId = p.regionId
		describeImagesRequest.ImageId = imageId
		ecsCli := p.clients.GetECSClient()
		describeImagesRes, err := ecsCli.DescribeImages(describeImagesRequest)
		utils.E("DescribeImages", err)
		if describeImagesRes.Images.Image[0].Status == "Available" {
			image = describeImagesRes.Images.Image[0]
			return image, nil
		}
		time.Sleep(time.Second * 5)
	}
	return image, fmt.Errorf("createImage timeout, imageId: %s", imageId)
}
