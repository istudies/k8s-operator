/*******************************************************************************
 * @File: main.go
 * @Description:
 * @Author: jiangxunyu
 * @Version: 1.0.0
 * @Date: 2022/8/12 11:34
*******************************************************************************/

// See https://github.com/kubernetes/sample-controller

package main

import (
	"context"
	"fmt"
	"log"

	clientset "github.com/istudies/k8s-operator/code-gen-crds/pkg/generated/clientset/versioned"
	"github.com/istudies/k8s-operator/code-gen-crds/pkg/generated/informers/externalversions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(err)
	}

	// get code-generator client sets
	client, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	// list my crd
	crdList, err := client.MycrdsV1().Foos("default").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range crdList.Items {
		// print
		fmt.Printf("kind: %s, name: %s\n", item.Kind, item.Name)
	}

	// generate custom informers
	factory := externalversions.NewSharedInformerFactory(client, 0)
	factory.Mycrds().V1().Foos().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			fmt.Printf("Add MyCrds Event... %s\n", key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, _ := cache.MetaNamespaceKeyFunc(newObj)
			fmt.Printf("Update MyCrds Event... %s\n", key)
		},
		DeleteFunc: func(obj interface{}) {
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			fmt.Printf("Delete MyCrds Event... %s\n", key)
		},
	})

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	<-stopCh
}
