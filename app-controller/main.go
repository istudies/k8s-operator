/*******************************************************************************
 * @File: main.go
 * @Description:
 * @Author: jiangxunyu
 * @Version: 1.0.0
 * @Date: 2022/8/12 17:41
*******************************************************************************/

package main

import (
	"log"
	"time"

	"github.com/istudies/k8s-operator/app-controller/pkg/controller"
	appclient "github.com/istudies/k8s-operator/app-controller/pkg/generated/clientset/versioned"
	"github.com/istudies/k8s-operator/app-controller/pkg/generated/informers/externalversions"
	"k8s.io/client-go/informers"
	internalclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// out-of-cluster
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		// in-cluster
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalln(err)
		}
	}
	// get built-in client
	internalClient, err := internalclient.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}
	// get custom generate client
	appClient, err := appclient.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}
	// new shared informer factory
	internalFactory := informers.NewSharedInformerFactory(internalClient, time.Second*30)
	appFactory := externalversions.NewSharedInformerFactory(appClient, time.Second*30)

	// construct app controller
	appController := controller.NewAppController(
		internalClient,
		appClient,
		internalFactory.Apps().V1().Deployments(),
		internalFactory.Core().V1().Services(),
		internalFactory.Networking().V1().Ingresses(),
		appFactory.Appcontroller().V1().Apps(),
	)

	// start shared informer factory
	stopCh := make(chan struct{})
	internalFactory.Start(stopCh)
	appFactory.Start(stopCh)

	internalFactory.WaitForCacheSync(stopCh)
	appFactory.WaitForCacheSync(stopCh)

	// run app controller
	appController.Run(5, stopCh)
}
