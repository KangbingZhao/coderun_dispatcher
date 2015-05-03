# coderun_dispatcher
the load balancing  module of code run , implemented by golang 




API设计
	/api/dispatcher/v1.0/image/:image-name
		返回json（server-ip,server-port）


最终的算法设计：
	思路： 先查容器，容器不过载则分配；容器过载继续下一个；若无可用容器则新建容器
	功能分拆：
		分配容器：
		维护状态：
