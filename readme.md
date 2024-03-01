# alyun-ops 帮助文档
如果你想以超低成本使用云服务器, 这个项目可以帮助你实现.
快速部署一个基于custom runtime 的 Golang Event 类型的 `aliyun抢占式实例自动维护` 到阿里云函数计算.
使用免费的aliyun serverless 自动维护低价抢占式实例.

## 前期准备
使用该项目，推荐您拥有以下的产品权限 / 策略：

| 服务/业务 | 函数计算 |     
| --- |  --- |   
| 权限/策略 | AliyunFCFullAccess |  

## 前提
- 安装git
- 安装serverless devs. 通过 [Serverless Devs Cli](https://www.serverless-devs.com/serverless-devs/install) 进行部署：
    - [安装 Serverless Devs Cli 开发者工具](https://www.serverless-devs.com/serverless-devs/install) ，并进行[授权信息配置](https://www.serverless-devs.com/fc/config) ；

> 注意: s deploy 之前的 actions 中 pre-deploy 中完成了编译， 如果编译过程中 go mod 下载很慢，可以考虑使用国内 go proxy 代理 [https://goproxy.cn/](https://goproxy.cn/)


## 使用
1. git clone 本项目
2. 安装serverless devs
3. 在 `code/asset/`目录下创建`.env`文件, 内容如下. 这是本项目调用aliyun ECS相关API的配置
```
aliyunRegionID = <aliyun regionId>
aliyunAccessKeyID = <aliyun accessKeyId>
aliyunAccessKeySecret = <aliyun accessKeySecret>
```
4. 在aliyun开通serverless FC
5. 在项目录下执行 `s deploy -y` 部署项目到serverlss FC
6. 在aliyun `云监控 - 事件监控 - 系统事件 - 事件报警规则`中创建规则, 填写表单如下:
  * 产品类型:ecs
  * 事件类型:状态通知
  * 事件等级:警告
  * 事件名称:抢占式实例中断通知
  * 勾选`函数计算`, 选择刚才部署的函数

## 如何本地调试
直接根据您的平台完成编译， 然后将目标二进制运行起来， 其实本质是启动了一个 http server，然后对这个  http server 发动 http 请求即可

**build**

```bash
$ cd code

# linux
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o target/main main.go

# mac
$ GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o target/main main.go

# windows
$ GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o target/main main.go
```

**debug**

``` bash
# 打开一个终端， 运行 target/main
# 然后打开另外一个终端，curl 发 http 请求
$ curl 127.0.0.1:9000/invoke -d "my event" -H "x-fc-request-id:rid123456"
```

## 测试

云监控 事件监控 抢占式实例终端通知事件数据样例
```json
{
    "product": "ECS",
    "resourceId": "acs:ecs:us-east-1:1537742922454562:instance/<resource-id>",
    "level": "WARN",
    "instanceName": "uk-prod006",
    "regionId": "eu-west-1",
    "groupId": "0",
    "name": "Instance:PreemptibleInstanceInterruption",
    "content": {
        "instanceId": "i-d7o6oaa1z4lpi4ekuf6k",
        "action": "de***"
    },
    "status": "Normal"
}
```

本地测试请求样例
```shell 
curl -X POST http://localhost:9000 -d '{"product":"ECS","resourceId":"acs:ecs:us-east-1:1537742922454562:instance/<resource-id>","level":"WARN","instanceName":"uk-prod006","regionId":"eu-west-1","groupId":"0","name":"Instance:PreemptibleInstanceInterruption","content":{"instanceId":"i-d7o6oaa1z4lpi4ekuf6k","action":"de***"},"status":"Normal"}'
```


## 测试用例

实例释放, 只有4会触发通知,且会自动购买新实例. 用于测试aliyun云监控通知事件触发.
1. 按量付费实例释放
2. 按量付费实例停机
3. 抢占式实例手动释放
4. 抢占式实例自动释放 (通过API模拟)
5. 抢占式实例停机

创建镜像
* 通过API模式抢占式实例自动释放, 触发创建镜像. (测试结束后,删除该镜像)

探测可用实例规格
* 有库存的实例规格
* 无库存的实例, 有多个相同内存, core不同
* 无库存的实例, 有多个相同内存, core相同

创建镜像
* 查询可用镜像ID, 随便选取一个, 创建实例
