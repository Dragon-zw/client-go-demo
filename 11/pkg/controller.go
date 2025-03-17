package pkg

import (
	"context"
	v14 "k8s.io/api/core/v1"
	v12 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informer "k8s.io/client-go/informers/core/v1"
	netInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	coreLister "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"time"
)

const (
	workNum  = 5
	maxRetry = 10
)

type controller struct {
	client        kubernetes.Interface
	ingressLister v1.IngressLister
	serviceLister coreLister.ServiceLister
	queue         workqueue.RateLimitingInterface
}

func (c *controller) updateService(oldObj interface{}, newObj interface{}) {
	// TODO: 比较 Annotation 是否相同
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *controller) addService(obj interface{}) {
	// TODO: 可以直接将 Object 放到 Queue
	c.enqueue(obj)
}

// 定义通用的方法
func (c *controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	// 把 Key 放在 Queue 当中(普通队列的方法)
	c.queue.Add(key)
}

func (c *controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v12.Ingress)
	// 通过 Ingress 获取到 Service
	ownerReference := v13.GetControllerOf(ingress)
	// 判断 Service 是否存在
	if ownerReference == nil {
		return
	}
	// 判断是否是 Service 的资源对象
	if ownerReference.Kind != "Service" {
		return
	}
	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

func (c *controller) Run(stopCh chan struct{}) {
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

// worker() 不断在 WorkQueue 中获取 Key 然后进行处理
func (c *controller) worker() {
	for c.processNextItem() {

	}
}

// 从 Queue 将 Key 获取出来然后处理
func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	// 将 Key 移除
	defer c.queue.Done(item)

	key := item.(string)

	// 调谐资源的状态
	err := c.syncService(key)
	if err != nil {
		c.handlerError(key, err)
	}
	return true
}

func (c *controller) syncService(key string) error {
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	// ############################################################
	// 删除
	// ############################################################
	service, err := c.serviceLister.Services(namespaceKey).Get(name)
	// 判断 Service 是否存在
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// ############################################################
	// 新增和删除
	// ############################################################
	_, ok := service.GetAnnotations()["ingress/http"] // 判断 "ingress/http" 是否存在
	ingress, err := c.ingressLister.Ingresses(namespaceKey).Get(name)
	if err != nil && errors.IsNotFound(err) {
		return err // 返回错误
	}

	if ok && errors.IsNotFound(err) {
		// Create ingress
		ig := c.constructIngress(service) // 创造 constructIngress 来重构 Ingress
		_, err := c.client.NetworkingV1().Ingresses(namespaceKey).Create(context.TODO(), ig, v13.CreateOptions{})
		if err != nil {
			return err
		}
	} else if !ok && ingress != nil {
		// Delete ingress
		err := c.client.NetworkingV1().Ingresses(namespaceKey).Delete(context.TODO(), name, v13.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	// 默认返回 nil 错误
	return nil
}

// TODO: 进行错误的处理(Error 是必须存在的)
func (c *controller) handlerError(key string, err error) {
	// 重试次数
	if c.queue.NumRequeues(key) <= maxRetry {
		c.queue.AddRateLimited(key)
		return
	}

	runtime.HandleError(err)
	c.queue.Forget(key)
}

// TODO: 创建 Ingress 资源对象
func (c *controller) constructIngress(service *v14.Service) *v12.Ingress {
	// 该资源用代码定义嵌套比较深，所以运维写Yaml配置文件比较直观
	// 可以改成读 Yaml 或者读接口，但是最终还是要改成这个数据格式
	ingress := v12.Ingress{}

	ingress.ObjectMeta.OwnerReferences = []v13.OwnerReference{
		*v13.NewControllerRef(service, v13.SchemeGroupVersion.WithKind("Service")),
	}

	ingress.Name = service.Name
	ingress.Namespace = service.Namespace
	pathType := v12.PathTypePrefix
	icn := "nginx"
	ingress.Spec = v12.IngressSpec{
		IngressClassName: &icn,
		Rules: []v12.IngressRule{
			{
				Host: "example.com",
				IngressRuleValue: v12.IngressRuleValue{
					HTTP: &v12.HTTPIngressRuleValue{
						Paths: []v12.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: v12.IngressBackend{
									Service: &v12.IngressServiceBackend{
										Name: service.Name,
										Port: v12.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return &ingress
}

// 工厂函数是面向对象的一种编程模式
func NewController(client kubernetes.Interface, serviceInformer informer.ServiceInformer, ingressInformer netInformer.IngressInformer) controller {
	// Controller 拥有了 Queue
	c := controller{
		client:        client,
		ingressLister: ingressInformer.Lister(),
		serviceLister: serviceInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManager"),
	}
	// 这就可以将 controller 逻辑和 Informer 逻辑结合
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})

	return c
}
