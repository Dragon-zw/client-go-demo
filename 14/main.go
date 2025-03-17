package main

import (
	v1 "baiding.tech/pkg/apis/baiding.tech/v1"
	"context"
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(err)
	}

	config.APIPath = "/apis/"
	config.NegotiatedSerializer = v1.Codecs.WithoutConversion()
	config.GroupVersion = &v1.GroupVersion

	client, err := rest.RESTClientFor(config)
	if err != nil {
		log.Fatalln(err)
	}

	foo := v1.Foo{}
	err = client.Get().Namespace("default").Resource("foos").Name("test-crd").Do(context.TODO()).Into(&foo)
	if err != nil {
		log.Fatalln(err)
	}

	// 调用 DeepCopy() 方法
	deepCopy := foo.DeepCopy()
	deepCopy.Spec.Name = "test2"

	fmt.Println(foo.Spec)
	fmt.Println(foo.Spec.Name)
	fmt.Println(foo.Spec.Replicas)

	fmt.Println(deepCopy.Spec.Name)
}
