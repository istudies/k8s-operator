/*******************************************************************************
 * @File: controller.go
 * @Description:
 * @Author: jiangxunyu
 * @Version: 1.0.0
 * @Date: 2022/8/11 16:54
*******************************************************************************/

package pkg

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	svc "k8s.io/api/core/v1"
	ing "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	svcInformer "k8s.io/client-go/informers/core/v1"
	ingINformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	svcLister "k8s.io/client-go/listers/core/v1"
	ingLister "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	workNum = 5
)

var once sync.Once

type controller struct {
	client        kubernetes.Interface
	queue         workqueue.RateLimitingInterface
	serviceLister svcLister.ServiceLister
	ingressLister ingLister.IngressLister
}

func (c *controller) addService(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	fmt.Println("add service...", key)
	// enqueue
	c.enqueue(key)
}

func (c *controller) updateService(oldObj interface{}, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		return
	}
	fmt.Println("update service...", key)
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	// enqueue
	c.enqueue(key)
}

func (c *controller) deleteIngress(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	fmt.Println("delete ingress...", key)
	ingress := obj.(*ing.Ingress)
	ownerReference := v1.GetControllerOf(ingress)
	if ownerReference == nil || ownerReference.Kind != "Service" {
		return
	}
	// enqueue
	c.enqueue(key)
}

func (c *controller) enqueue(key string) {
	c.queue.Add(key)
}

func (c *controller) Run(stopCh <-chan struct{}) {
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

func (c *controller) worker() {
	for {
		if !c.workProcessNextKey() {
			fmt.Println("work process failed")
		}
	}
}

func (c *controller) workProcessNextKey() bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)
	err := c.syncService(key.(string))
	if err != nil {
		c.handleErr(key.(string), err)
	}
	return true
}

func (c *controller) syncService(key string) error {
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	// get service
	service, err := c.serviceLister.Services(namespaceKey).Get(name)
	if errors.IsNotFound(err) {
		// delete service
		_ = c.client.CoreV1().Services(namespaceKey).Delete(context.TODO(), name, v1.DeleteOptions{})
		return nil
	}
	if err != nil {
		return err
	}
	// check annotations
	v, ok := service.Annotations["ingress/http"]
	// get ingress
	ingress, err := c.ingressLister.Ingresses(namespaceKey).Get(name)
	if ok && v == "true" && errors.IsNotFound(err) {
		// create ingress
		newIng := c.constructIngress(service)
		_, err := c.client.NetworkingV1().Ingresses(namespaceKey).Create(context.TODO(), newIng, v1.CreateOptions{})
		if err != nil {
			return err
		}
		fmt.Println("create ingress success.")
	} else if (!ok || v == "false") && ingress != nil {
		// delete ingress
		err := c.client.NetworkingV1().Ingresses(namespaceKey).Delete(context.TODO(), name, v1.DeleteOptions{})
		if err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func (c *controller) handleErr(key string, err error) {
	if c.queue.NumRequeues(key) <= 10 {
		c.queue.AddRateLimited(key)
		return
	}
	c.queue.Forget(key)
	runtime.HandleError(err)
}

func (c *controller) constructIngress(service *svc.Service) *ing.Ingress {
	ingress := ing.Ingress{}
	ingress.Namespace = service.Namespace
	ingress.Name = service.Name
	ingress.OwnerReferences = service.GetOwnerReferences()
	pathType := ing.PathTypePrefix
	ingressClassName := "nginx"
	ingress.Spec = ing.IngressSpec{
		IngressClassName: &ingressClassName,
		Rules: []ing.IngressRule{
			{
				Host: "testing.com",
				IngressRuleValue: ing.IngressRuleValue{
					HTTP: &ing.HTTPIngressRuleValue{
						Paths: []ing.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: ing.IngressBackend{
									Service: &ing.IngressServiceBackend{
										Name: service.Name,
										Port: ing.ServiceBackendPort{
											Number: service.Spec.Ports[0].Port,
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

func NewController(client kubernetes.Interface, serviceInformer svcInformer.ServiceInformer, ingressInformer ingINformer.IngressInformer) *controller {
	var ctl controller
	once.Do(func() {
		ctl = controller{
			client:        client,
			queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "testing"),
			serviceLister: serviceInformer.Lister(),
			ingressLister: ingressInformer.Lister(),
		}

		// event handler
		serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.addService,
			UpdateFunc: ctl.updateService,
		})

		ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: ctl.deleteIngress,
		})
	})
	return &ctl
}
