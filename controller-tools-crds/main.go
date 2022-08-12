/*******************************************************************************
 * @File: main.go
 * @Description:
 * @Author: jiangxunyu
 * @Version: 1.0.0
 * @Date: 2022/8/12 14:24
*******************************************************************************/

package main

import (
	"context"
	"fmt"
	"log"

	v1 "github.com/istudies/k8s-operator/controller-tools-crds/pkg/apis/ctl.mycrds.com/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(config)
	}
	config.APIPath = "/apis"
	config.GroupVersion = &v1.GroupVersion
	config.NegotiatedSerializer = v1.Codec.WithoutConversion()

	client, err := rest.RESTClientFor(config)
	if err != nil {
		log.Fatalln(config)
	}

	mycrd := v1.MyCRD{}
	err = client.Get().Namespace("default").
		Resource("mycrds").Name("mycrd-by-ctl").
		Do(context.TODO()).Into(&mycrd)
	if err != nil {
		log.Fatalln(err)
	}
	// load mycrd name: mycrd-by-ctl, replicas: 3
	fmt.Printf("load mycrd name: %s, replicas: %d\n", mycrd.Spec.Name, mycrd.Spec.Replicas)
}
