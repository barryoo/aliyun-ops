package ecs

import (
	"log"
	"testing"

	"businessmatics.io/aliyun-ops/aliyun"
	"businessmatics.io/aliyun-ops/asset"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	gr "github.com/awesome-fc/golang-runtime"
	"github.com/driftprogramming/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	err := godotenv.Load(asset.Env, ".env")
	if err != nil {
		log.Printf("Error loading .env file, err: %s", err)
		panic(err)
	}
}

func TestDetectInstanceTypes(t *testing.T) {

	processor := NewProcessor(&gr.FCContext{}, "eu-west-1", "i-d7o896vy5aehfawi7pwi")
	processor.fcLogger = gr.GenLoggerByRequestID("1")
	processor.clients = aliyun.Clients{}

	instance := ecs.Instance{}
	instance.RegionId = "eu-west-1"
	instance.ZoneId = "eu-west-1a"

	//ecs.g5.large 无库存的实例, 有多个相同内存, core相同
	instance.InstanceType = "ecs.g5.large"
	instance.Memory = 8 * 1024
	instance.Cpu = 2

	//ecs.g5.large 无库存的实例, 有多个相同内存, core不同
	instance.InstanceType = "ecs.t5-c1m4.xlarge"
	instance.Memory = 16 * 1024
	instance.Cpu = 4

	logrus.Infof("finding suitable InstanceType for instance: %v", instance)
	suitableResource := processor.detectInstanceTypes(&instance)
	if &suitableResource == nil {
		assert.Fail(t, "can't find suitable instance")
	} else {
		logrus.Infof("found suitable InstanceType: %v", suitableResource)
	}
}

func TestQueryImageStatus(t *testing.T) {
	imageId := "m-d7o2t5141dlkx0j16xz6"

	processor := NewProcessor(&gr.FCContext{}, "eu-west-1", "i-d7o896vy5aehfawi7pwi")
	processor.fcLogger = gr.GenLoggerByRequestID("1")
	processor.clients = aliyun.Clients{}

	processor.describeImageStatus(imageId)
}
