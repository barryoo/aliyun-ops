package main

import (
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

// Clients 用户管理所有的aliyun sdk client.
type Clients struct {
	ecsClient *ecs.Client // ECS云服务器客户端
	mutex     sync.Mutex  // 锁
}

// GetECSClient 获取ECS云服务器客户端，此方法线程安全。
func (m *Clients) GetECSClient() *ecs.Client {
	if m.ecsClient == nil {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		if m.ecsClient == nil {
			// 创建ECS云服务器客户端
			ecsClient, err := ecs.NewClientWithAccessKey(RegionId(), AccessKeyID(), AccessKeySecret())
			if err != nil {
				panic(err)
			}
			m.ecsClient = ecsClient
		}
	}
	return m.ecsClient
}
