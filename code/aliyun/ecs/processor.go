package ecs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"

	"businessmatics.io/aliyun-ops/aliyun"
	"businessmatics.io/aliyun-ops/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	gr "github.com/awesome-fc/golang-runtime"
	"github.com/sirupsen/logrus"
)

const (
	SpotStrategy = "SpotAsPriceGo"
	dryRun       = false
)

type Processor struct {
	// contains filtered or unexported fields
	ctx        *gr.FCContext
	fcLogger   *logrus.Entry
	regionId   string
	instanceId string
	clients    aliyun.Clients
}

func NewProcessor(ctx *gr.FCContext, regionId string, instanceId string) *Processor {
	return &Processor{
		ctx:        ctx,
		regionId:   regionId,
		instanceId: instanceId,
		fcLogger:   ctx.GetLogger(),
		clients:    aliyun.Clients{},
	}
}

func (p *Processor) Process() ([]byte, error) {
	p.fcLogger.Infof(" resionId: %s, instanceId: %s", p.regionId, p.instanceId)
	instance := p.describeInstance()
	p.describeInstanceStatus()

	//查询可用区的实例规格
	suitableResource := p.detectInstanceTypes(&instance)
	if suitableResource.instanceType != "" {
		return nil, fmt.Errorf("no suitable resource")
	}
	p.fcLogger.Infof("found suitable resource: %v", suitableResource)

	//查询存储信息
	disk := p.describeDisks(&instance)
	p.fcLogger.Infof("describe disk info: diskSize:%d, diskType:%s", disk.Size, disk.Type)

	//todo test 测试时注释掉, 发布时打开
	imageId := p.createImage(instance)
	p.fcLogger.Infof("imageId: %s", imageId)
	_, err := p.describeImageStatus(imageId)
	utils.P("describeImageStatus", err)

	//创建新的抢占式实例
	createInstanceId := p.createNewInstance(&instance, imageId, suitableResource, disk)
	if createInstanceId == "" {
		return nil, fmt.Errorf("create new instance failed")
	}

	return []byte("success"), nil
}

// 查询即将被释放的实例的信息
func (p *Processor) describeInstance() ecs.Instance {
	instancesRequest := ecs.CreateDescribeInstancesRequest()
	instancesRequest.RegionId = p.regionId
	instanceIdByte, err := json.Marshal([]string{p.instanceId})
	utils.P("Marshal DescribeInstancesRequest.InstanceIds", err)
	instancesRequest.InstanceIds = string(instanceIdByte)
	res, err := p.clients.GetECSClient().DescribeInstances(instancesRequest)
	utils.P("DescribeInstances", err)
	if res.Instances.Instance[0].OperationLocks.LockReason != nil {
		// 打印所有的锁定原因 LockReason: financial：因欠费被锁定。 security：因安全原因被锁定。Recycling：抢占式实例的待释放锁定状态。 dedicatedhostfinancial：因为专有宿主机欠费导致ECS实例被锁定。 refunded：因退款被锁定。
		for _, lockReason := range res.Instances.Instance[0].OperationLocks.LockReason {
			p.fcLogger.Infof("lockReason: %s", lockReason)
		}
	}
	return res.Instances.Instance[0]
}

// 查询即将被释放的实例的状态
func (p *Processor) describeInstanceStatus() ecs.InstanceStatuses {
	instanceStatusRequest := ecs.CreateDescribeInstanceStatusRequest()
	instanceStatusRequest.RegionId = p.regionId
	instanceStatusRequest.InstanceId = &([]string{p.instanceId})
	ecsCli := p.clients.GetECSClient()
	instanceStatusRes, err := ecsCli.DescribeInstanceStatus(instanceStatusRequest)
	utils.P("DescribeInstanceStatus", err)
	instanceStatuses := instanceStatusRes.InstanceStatuses
	//打印所有的实例状态.
	for _, instanceStatus := range instanceStatusRes.InstanceStatuses.InstanceStatus {
		p.fcLogger.Infof("instanceStatus: %s", instanceStatus)
	}
	return instanceStatuses
}

// 基于即将被释放的实例,创建镜像
func (p *Processor) createImage(instance ecs.Instance) string {
	createImageRequest := ecs.CreateCreateImageRequest()
	createImageRequest.RegionId = instance.RegionId
	createImageRequest.InstanceId = instance.InstanceId
	createImageRequest.ImageName = instance.InstanceName
	ecsCli := p.clients.GetECSClient()
	createImageRes, err := ecsCli.CreateImage(createImageRequest)
	utils.P("CreateImage", err)
	return createImageRes.ImageId
}

// 查询镜像的状态, 循环查询,直到镜像状态为Available, 则表示镜像创建成功, 否则等待5秒,继续查询镜像状态. 最多重试12次
func (p *Processor) describeImageStatus(imageId string) (image ecs.Image, err error) {
	for i := 0; i < 20; i++ {
		describeImagesRequest := ecs.CreateDescribeImagesRequest()
		describeImagesRequest.RegionId = p.regionId
		describeImagesRequest.ImageId = imageId
		ecsCli := p.clients.GetECSClient()
		describeImagesRes, err := ecsCli.DescribeImages(describeImagesRequest)
		utils.P("DescribeImages", err)
		if len(describeImagesRes.Images.Image) > 0 && describeImagesRes.Images.Image[0].Status == "Available" {
			image = describeImagesRes.Images.Image[0]
			return image, nil
		}
		time.Sleep(time.Second * 10)
	}
	return image, fmt.Errorf("createImage timeout, imageId: %s", imageId)
}

/*
探测与当前实例规格最接近的实例规格
1. 查询与当前实例一致的规格的库存
2. 如果无库存, 则根据当前实例的内存, 查询所有有库存的规格. 然后选择价格最低(cpu最小)的规格
*/
func (p *Processor) detectInstanceTypes(instance *ecs.Instance) (suit suitableResource) {
	//1. 查询与当前实例一致的规格的库存, 抢占式, 系统自动出价
	req := ecs.CreateDescribeAvailableResourceRequest()
	req.RegionId = instance.RegionId
	req.InstanceType = instance.InstanceType
	req.SpotStrategy = "SpotAsPriceGo"
	req.DestinationResource = "InstanceType"
	suits := p.findSuitableAvailableResource(req)
	if len(suits) > 0 {
		//遍历suits,优先找同zoneId同instanceType的, 如果没有,则优先找同instanceType. 如果还没有,则放弃
		var sameInstanceTypeSuit *suitableResource
		for _, s := range suits {
			if s.instanceType == instance.InstanceType {
				sameInstanceTypeSuit = &s
				if s.zoneId == instance.ZoneId {
					suit = s
					return
				}
			}
		}
		if sameInstanceTypeSuit.instanceType != "" {
			suit = *sameInstanceTypeSuit
			return
		}
	}

	//2.如果没有找到可用库存, 则根据当前实例规格的内存, 查询可用库存.
	req = ecs.CreateDescribeAvailableResourceRequest()
	req.RegionId = instance.RegionId
	req.SpotStrategy = "SpotAsPriceGo"
	req.DestinationResource = "InstanceType"
	req.Memory = requests.NewFloat(float64(instance.Memory / 1024))

	suits = p.findSuitableAvailableResource(req)
	//查询具体配置,cpu memory
	suits = p.queryInstanceTypeDetail(suits)
	if len(suits) > 0 {
		//遍历suits
		//找同memory的, 如果有, 则按照cpu排序, 优先cpu小的 zone相同的.  如果没有同memory的, 则放弃
		var sameMemorySuit suitableResource
		for _, s := range suits {
			if s.core != 0 && s.memory == float64(instance.Memory/1024) {
				if sameMemorySuit.instanceType == "" {
					sameMemorySuit = s
				} else if s.core < sameMemorySuit.core {
					sameMemorySuit = s
				} else if s.core == sameMemorySuit.core && s.zoneId == instance.ZoneId {
					sameMemorySuit = s
				}
			}
		}
		if sameMemorySuit.instanceType != "" {
			suit = sameMemorySuit
			return
		}
	}

	return
}

func (p *Processor) findSuitableAvailableResource(request *ecs.DescribeAvailableResourceRequest) (suits []suitableResource) {
	p.fcLogger.Infof("finding best available resource for request: %v", request)
	res, err := p.clients.GetECSClient().DescribeAvailableResource(request)
	utils.P("detect available of same instanceType", err)
	//遍历,找到可用的zone. 优先匹配当前zone, 如果当前zone无可用的, 则选择其他zone.
	for _, az := range res.AvailableZones.AvailableZone {
		for _, availableResource := range az.AvailableResources.AvailableResource {
			for _, supportedResource := range availableResource.SupportedResources.SupportedResource {
				if supportedResource.Status != "Available" {
					continue
				}
				suits = append(suits, suitableResource{zoneId: az.ZoneId, instanceType: supportedResource.Value})
			}
		}
	}
	return
}

func (p *Processor) queryInstanceTypeDetail(suits []suitableResource) []suitableResource {
	if len(suits) == 0 {
		return suits
	}
	req := ecs.CreateDescribeInstanceTypesRequest()
	req.RegionId = p.regionId
	var instanceTypes []string
	for _, s := range suits {
		instanceTypes = append(instanceTypes, s.instanceType)
	}
	req.InstanceTypes = &instanceTypes
	res, err := p.clients.GetECSClient().DescribeInstanceTypes(req)
	utils.P("query instanceType detail", err)
	instanceTypeIdMap := make(map[string]ecs.InstanceType)
	for i := 0; i < len(res.InstanceTypes.InstanceType); i++ {
		it := res.InstanceTypes.InstanceType[i]
		instanceTypeIdMap[it.InstanceTypeId] = it
	}

	for i := 0; i < len(suits); i++ {
		s := &suits[i]
		it := instanceTypeIdMap[s.instanceType]
		s.core = it.CpuCoreCount
		s.memory = it.MemorySize
	}
	return suits
}

func (p *Processor) createNewInstance(instance *ecs.Instance, imageId string, suitableResource suitableResource, disk ecs.Disk) (createInstanceId string) {
	req := ecs.CreateRunInstancesRequest()
	req.DryRun = requests.NewBoolean(dryRun)
	//基本信息
	req.RegionId = instance.RegionId
	req.ZoneId = suitableResource.zoneId
	req.ImageId = imageId
	req.InstanceType = suitableResource.instanceType
	req.InstanceName = instance.InstanceName
	req.HostName = instance.HostName
	req.SpotStrategy = SpotStrategy
	req.SpotDuration = requests.NewInteger(0)
	//网络
	req.VSwitchId = instance.VpcAttributes.VSwitchId
	req.InternetChargeType = instance.InternetChargeType
	if instance.InternetMaxBandwidthIn > 0 {
		req.InternetMaxBandwidthIn = requests.NewInteger(instance.InternetMaxBandwidthIn)
	}
	if instance.InternetMaxBandwidthOut > 0 {
		req.InternetMaxBandwidthOut = requests.NewInteger(instance.InternetMaxBandwidthOut)
	}
	//安全
	req.PasswordInherit = requests.NewBoolean(true)
	req.CreditSpecification = instance.CreditSpecification
	req.SecurityGroupIds = &(instance.SecurityGroupIds.SecurityGroupId)
	//磁盘
	req.SystemDiskSize = strconv.Itoa(disk.Size)
	req.SystemDiskCategory = "cloud_efficiency"
	p.fcLogger.Infof("create new instance with request: %v", req)
	res, err := p.clients.GetECSClient().RunInstances(req)
	utils.P("create and run instance ", err)
	createInstanceId = res.InstanceIdSets.InstanceIdSet[0]
	return
}

func (p *Processor) describeDisks(instance *ecs.Instance) (disk ecs.Disk) {
	req := ecs.CreateDescribeDisksRequest()
	req.RegionId = instance.RegionId
	req.InstanceId = instance.InstanceId
	res, err := p.clients.GetECSClient().DescribeDisks(req)
	utils.P("describe disks", err)
	for _, d := range res.Disks.Disk {
		if d.Type == "system" {
			disk = d
		}
	}
	return
}

type suitableResource struct {
	zoneId       string
	instanceType string
	core         int
	memory       float64
}
