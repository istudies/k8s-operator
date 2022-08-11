/*******************************************************************************
 * @File: main.go
 * @Description:
 * @Author: jiangxunyu
 * @Version: 1.0.0
 * @Date: 2022/8/10 13:19
*******************************************************************************/

package main

import (
	"log"

	"github.com/istudies/k8s-operator/builtin-k8s-controller/pkg"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalln(err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	// delta FIFO Queue
	// queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	//factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace("default"))
	factory := informers.NewSharedInformerFactory(clientset, 0)
	//lister := factory.Core().V1().Pods().Lister()
	//factory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	//	AddFunc: func(obj interface{}) {
	//		key, _ := cache.MetaNamespaceKeyFunc(obj)
	//		fmt.Printf("ADD Event... key: %s\n", key)
	//	},
	//	UpdateFunc: func(oldObj, newObj interface{}) {
	//		key, _ := cache.MetaNamespaceKeyFunc(newObj)
	//		fmt.Printf("Update Event... key: %s\n", key)
	//
	//	},
	//	DeleteFunc: func(obj interface{}) {
	//		key, _ := cache.MetaNamespaceKeyFunc(obj)
	//		fmt.Printf("Delete Event... key: %s\n", key)
	//	},
	//})

	ctl := pkg.NewController(clientset, factory.Core().V1().Services(), factory.Networking().V1().Ingresses())

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	ctl.Run(stopCh)
}
