package main

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func main() {

	// RESTClient
	/*// Step1: config
	// 在 client-go 中提供了一些方法可以帮助我们创建 config 对象（跟 APIServe 进行交互），config 对象可以从
	// 1.指定的配置文件里面获取
	// 2.集群内部里面获取（ServiceAccount）挂载的文件进行获取

	// 从配置文件中获取（clientcmd.BuildConfigFromFlags可以获取 config 对象）
	// masterURL 如果为空，那么就是从 kubeconfig 文件获取
	// clientcmd.RecommendedHomeFile 使用默认的配置文件的路径获取
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	// 对错误进行处理
	if err != nil {
		panic(err)
	}

	// Step2: client
	config.GroupVersion = &v1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs
	config.APIPath = "/api"

	// 通过 rest.RESTClientFor 的方法创建 client
	restClient, err := rest.RESTClientFor(config)
	// 对错误进行处理
	if err != nil {
		panic(err)
	}

	// Step3: get data
	// Name() 指定获取的资源的名称，例如获取 test Pod 的资源对象
	// Do() 方法会实际调用 API
	pod := v1.Pod{}
	err = restClient.Get().Namespace("default").Resource("pods").Name("test").Do(context.TODO()).Into(&pod)
	if err != nil {
		println(err)
	} else {
		println(pod.Name)
	}
	// 套路: config 获取 cli，cli 执行对应的方法。

	// 发送请求之后，我们可以拿到 result 的对象，提供 Into 方法和 Get 方法
	// Into 方法可以将结果我们指定的对象里面中
	// Get 方法直接从响应中获取原始的数据或对象。*/

	// ClientSet
	/*config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	// NewForConfig(传递 config) 可以获取 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	// 需要通过 clientset 创建指定资源的 Client，然后该 Client 来操作资源
	coreV1 := clientset.CoreV1()
	pod, err := coreV1.Pods("default").Get(context.TODO(), "test", v1.GetOptions{})
	if err != nil {
		println(err)
	} else {
		println(pod.Name)
	}
	// 总结：
	// 需要创建 clientset，然后 clientset 可以根据 API Group/Version 进行分组
	// 每一个 Group/Version 下都有对应的 Client，拿到这个 Client 之后就能通过调用相应的方法来拿到对应的资源
	// 再通过对应的资源提供的方法(Get, List, Delete等)操作对应的资源*/

	// dynamicClient
	/*// 加载 kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	// 创建动态客户端
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	// 定义 Pod 的 GVR
	gvr := schema.GroupVersionResource{
		Group:    "", // Core group
		Version:  "v1",
		Resource: "pods",
	}
	// 列出默认命名空间中的所有 Pods
	listResources(dynamicClient, gvr, "default")*/

	// DiscoveryClient
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	// 创建 DiscoveryClient
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err)
	}
	// 获取并打印服务器版本
	getServerVersion(discoveryClient)

	// 获取并打印所有 API 组
	getAPIGroups(discoveryClient)

	// 获取并打印所有 API 资源
	getAllAPIResources(discoveryClient)

	// 如果需要，可以获取特定组的资源
	// getAPIResourcesForGroup(discoveryClient, "apps/v1")

}

func listResources(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace string) {
	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing resources: %s", err.Error())
	}

	for _, item := range list.Items {
		fmt.Printf("Pod Name: %s\n", item.GetName())
	}
}

func getServerVersion(discoveryClient *discovery.DiscoveryClient) {
	version, err := discoveryClient.ServerVersion()
	if err != nil {
		log.Fatalf("Error fetching server version: %s", err.Error())
	}

	fmt.Printf("Server Version: %s\n", version.GitVersion)
}

func getAPIGroups(discoveryClient *discovery.DiscoveryClient) {
	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		log.Fatalf("Error fetching server groups: %s\n", err.Error())
	}

	for _, group := range groups.Groups {
		fmt.Printf("Group: %s\n", group.Name)
		for _, version := range group.Versions {
			fmt.Printf("  Version: %s\n", version.Version)
		}
	}
}

func getAllAPIResources(discoveryClient *discovery.DiscoveryClient) {
	resources, err := discoveryClient.ServerResources()
	if err != nil {
		log.Fatalf("Error fetching server resources: %s\n", err.Error())
	}

	for _, api := range resources {
		fmt.Printf("Group: %s, Version: %s\n", api.GroupVersion, api.APIVersion)
		for _, resource := range api.APIResources {
			fmt.Printf("  Resource: %s, Namespaced: %v\n", resource.Name, resource.Namespaced)
		}
	}
}
