# aliyun-ops

快速部署一个基于custom runtime 的 Golang Event 类型的 `aliyun抢占式实例自动维护` 到阿里云函数计算
当前仅支持aliyun serverless, 暂不支持其他云服务商.

## 功能
* 用来做什么的?
用来低成本使用aliyun ECS服务器.
aliyun ECS抢占式实例成本只有普通服务器的十分之一, 甚至更低.但是随时有被释放的风险.
本应用可以在抢占式实例被释放前, 自动复制出一套几乎一样的实例,并保留所有数据.
如果你把服务器中的服务设置为开启启动, 则可以确保服务不变.
由于当前还不支持外网IP的绑定, 所以如果你需要对外提供网络服务, 则需要有另外一台服务器做代理. 

* 什么原理?
抢占式实例被释放前5分钟, 会向`云监控`发送通知.
本项目监听通知, 根据通知中的内容, 自动复制出一套几乎一样的实例.

* 复制抢占式实例时, 会复制哪些配置?
  * 在aliyun服务器资源充足的情况下, CPU/内存规格尽量与原规格一致, 如果资源不足, 则会选择选择同内存的规格.
  * 基于原磁盘创建镜像, 基于镜像创建新的服务器实例. 所以数据完全保留.
  * 除了IP之外, 其他配置保持不变.

## 前期准备
使用该项目，推荐您拥有以下aliyun产品权限/策略：

| 服务/业务 | 函数计算 |     
| --- |  --- |   
| 权限/策略 | AliyunFCFullAccess |  

## 前提
- 安装git
- 安装serverless devs. 通过 [Serverless Devs Cli](https://www.serverless-devs.com/serverless-devs/install) 进行部署：
    - [安装 Serverless Devs Cli 开发者工具](https://www.serverless-devs.com/serverless-devs/install) ，并进行[授权信息配置](https://www.serverless-devs.com/fc/config)

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
6. 在aliyun `云监控 - 事件中心 - 事件订阅 - 创建订阅规则`中创建规则, 填写表单如下:
  * 产品类型:ecs
  * 事件类型:状态通知
  * 事件等级:警告
  * 事件名称:抢占式实例中断通知
  * 推送与集成, 创建推送渠道, 目标类型选择`函数计算`, 选择刚才部署的函数

## 本地构建与调试
直接根据您的平台完成编译， 然后将目标二进制运行起来， 其实本质是启动了一个 http server，然后对这个  http server 发动 http 请求即可

**build**

```bash
$ cd code

# 在linux平台, 编译linux平台的二进制
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o target/main main.go

# 在macos平台, 编译macos平台的二进制
$ GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o target/main main.go

# 在windows平台, 编译windows平台的二进制
$ GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o target/main main.go
```

**run**
先build, 然后运行 target/main文件. 运行成功后, 使用curl发起http请求来进行测试.
```bash
$ cd /code/target
$ ./main
$ curl 127.0.0.1:9000/invoke -d "my event" -H "x-fc-request-id:rid123456"
```

## 本地测试
本地测试,通过curl发送请求来进行.

云监控事件监控`抢占式实例`通知事件数据样例如下, 意思是,"uk-prod006"这个ECS实例即将释放. 
你可以根据自己的实际情况,修改数据, 用于测试.
```json
{
    "product": "ECS",
    "resourceId": "acs:ecs:cn-hangzhou:1342272449033182:instance/<resource-id>",
    "level": "WARN",
    "instanceName": "instanceName",
    "regionId": "cn-hangzhou",
    "name": "Instance:PreemptibleInstanceInterruption",
    "content": {
        "instanceId": "i-b***3m3",
        "instanceName": "wor***639",
        "action": "de***"
    },
    "status": "Normal"
}
```

本地测试CURL命令
```shell 
curl -X POST http://localhost:9000 -d '{"product":"ECS","resourceId":"acs:ecs:us-east-1:1537742922454562:instance/<resource-id>","level":"WARN","instanceName":"uk-prod006","regionId":"eu-west-1","groupId":"0","name":"Instance:PreemptibleInstanceInterruption","content":{"instanceId":"i-d7o6oaa1z4lpi4ekuf6k","action":"de***"},"status":"Normal"}'
```

## 云环境测试
函数在aliyun部署成功后,可以直接使用云环境进行测试.
在`云监控`中, 找到你创建的`事件订阅`, 点击`调试事件订阅`按钮, 选择需要的参数,


## 测试用例
本项目进行了以下几种测试, 代码见`/code/aliyun/ecs/Processor_test.go`

* 实例释放, 只有4会触发通知,且会自动购买新实例. 用于测试aliyun云监控通知事件触发.
  1. 按量付费实例释放
  2. 按量付费实例停机
  3. 抢占式实例手动释放
  4. 抢占式实例自动释放 (通过API模拟)
  5. 抢占式实例停机

* 创建镜像
  * 通过API模式抢占式实例自动释放, 触发创建镜像. (测试结束后,删除该镜像)

* 探测可用实例规格
  * 有库存的实例规格
  * 无库存的实例, 有多个相同内存, core不同
  * 无库存的实例, 有多个相同内存, core相同

* 创建镜像
  * 查询可用镜像ID, 随便选取一个, 创建实例
