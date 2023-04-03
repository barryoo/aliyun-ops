package main

import "os"

const (
	envVarsRegionId        = "aliyunRegionID"
	envVarsAccessKeyId     = "aliyunAccessKeyID"
	envVarsAccessKeySecrec = "aliyunAccessKeySecret"
)

func RegionId() string {
	regionId := os.Getenv(envVarsRegionId)
	if regionId == "" {
		panic("aliyunRegionID env is not set")
	} else {
		return regionId
	}
}

func AccessKeyID() string {
	accessKeyID := os.Getenv(envVarsAccessKeyId)
	if accessKeyID == "" {
		panic("aliyunAccessKeyID env is not set")
	} else {
		return accessKeyID
	}

}

func AccessKeySecret() string {
	accessKeySecret := os.Getenv(envVarsAccessKeySecrec)
	if accessKeySecret == "" {
		panic("aliyunAccessKeySecret env is not set")
	} else {
		return accessKeySecret
	}
}
