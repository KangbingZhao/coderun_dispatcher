# coderun_dispatcher
	the load balancing  module of code run , implemented by golang 




* API设计
	/api/dispatcher/v1.0/image/:image-name
		返回json（server-ip,server-port）


* 最终的算法设计：
	思路： 先查容器，容器不过载则分配；容器过载继续下一个；若无可用容器则新建容器
	功能分拆：
		分配容器：
		维护状态：

* 功能划分
1. Stats.go 更新集群状态
2. Cache.go 维护容器缓存
3. Algorithm.go 处理容器请求

## Stats.go说明
### getInitialServerAddr() 
	从 /metadata/config.json中读取服务器的配置信息,返回serverConfig
### getServerStats()   
	接受集群的地址，返回所有服务器的配置和负载信息。
		仅包含在线的服务器，不在线的会过滤
### getValidContainerName()
	获取一个服务器中所有容器的ID

### getContainerStats()
	获取单个服务器中所有容器的状态(总有一些容器docker api看不到)


##Algorithm.go说明
### isOverload()
	判断一个服务器是否过载
### createNewContainer()
	创建一个新容器
### RR()
	Round-Robin算法
### LCS()
	Least-Connection-Scheduling
### GetServerLoad()
	获取服务器负载
### ServerPriority()
	选择负载最低的服务器
### findImageInServer()
	查看一个服务器中是否有对应镜像的容器
### sortServerByLoad()
	按照负载从低到高对服务器排序
### ServerAndContainer()
	分配算法，若存在镜像，镜像和服务器都不超载则分配；若仅仅镜像超载则重建镜像；若服务器超载则新建容器
