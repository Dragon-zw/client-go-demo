package main

import (
	"github.com/Dragon-zw/client-go-demo/11/pkg"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func main() {
	/*
		1. config (需要通过 config 文件获取到 config 对象)
		2. client (通过 config 对象来创建 clientSet 来操作内置的资源对象)
		3. informer (通过 factory 来创建 informer)
		4. add event handler
		5. informer.Start
	*/
	// Step 1: config
	// 如果是运行集群外部获取 config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		// 如果是运行集群内部获取 config，则执行以下代码
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln("Can't get in cluster config")
		}
		config = inClusterConfig
	}

	// Step 2: client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln("Can't create clientset")
	}

	// Step 3: informer
	factory := informers.NewSharedInformerFactory(clientset, 0)
	serviceInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()

	// Step 4: add event handler
	// pkg/controller.go.bak
	// 已经将 informer/add event handler 交由 pkg/controller.go.bak 进行处理
	// defaultResync 参数：周期性的拉取全量对象保存到本地缓存中，之前的数据全部丢弃

	// Step 5: informer start
	// 需要提前传递参数
	controller := pkg.NewController(clientset, serviceInformer, ingressInformer)
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	// 等待所有的消息同步之后，再来执行 Run() 方法
	factory.WaitForCacheSync(stopCh)

	// 消耗队列里面事件
	controller.Run(stopCh)
}
