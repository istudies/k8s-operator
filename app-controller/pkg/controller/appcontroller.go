/*******************************************************************************
 * @File: appcontroller.go
 * @Description:
 * @Author: jiangxunyu
 * @Version: 1.0.0
 * @Date: 2022/8/12 17:42
*******************************************************************************/

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	appcontrollerv1 "github.com/istudies/k8s-operator/app-controller/pkg/apis/appcontroller/v1"
	appClient "github.com/istudies/k8s-operator/app-controller/pkg/generated/clientset/versioned"
	appInformer "github.com/istudies/k8s-operator/app-controller/pkg/generated/informers/externalversions/appcontroller/v1"
	applister "github.com/istudies/k8s-operator/app-controller/pkg/generated/listers/appcontroller/v1"
	deployapps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	deploymentInformer "k8s.io/client-go/informers/apps/v1"
	coreInformer "k8s.io/client-go/informers/core/v1"
	netInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	internalclient "k8s.io/client-go/kubernetes"
	deploylister "k8s.io/client-go/listers/apps/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	netlister "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const controllerBy = "controllerBy"

var once sync.Once

type appController struct {
	internalClient   internalclient.Interface
	appClient        appClient.Interface
	deploymentLister deploylister.DeploymentLister
	serviceLister    corelister.ServiceLister
	ingressLister    netlister.IngressLister
	appLister        applister.AppLister
	queue            workqueue.RateLimitingInterface
}

func NewAppController(internalClient *kubernetes.Clientset, appClient *appClient.Clientset,
	deployInformer deploymentInformer.DeploymentInformer, svcInformer coreInformer.ServiceInformer,
	ingInformer netInformer.IngressInformer, appInformer appInformer.AppInformer) *appController {
	var ctl appController
	once.Do(func() {
		ctl = appController{
			internalClient:   internalClient,
			appClient:        appClient,
			deploymentLister: deployInformer.Lister(),
			serviceLister:    svcInformer.Lister(),
			ingressLister:    ingInformer.Lister(),
			appLister:        appInformer.Lister(),
			queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "appControllerQueue"),
		}

		// event handler
		deployInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: ctl.deleteDeploymentEvent,
		})
		svcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: ctl.deleteSvcEvent,
		})
		ingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: ctl.deleteIngressEvent,
		})
		appInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.addAppEvent,
			UpdateFunc: ctl.updateAppEvent,
			DeleteFunc: ctl.deleteAppEvent,
		})
	})
	return &ctl
}

func (c *appController) Run(workerNum uint32, stopCh <-chan struct{}) {
	for i := 0; uint32(i) < workerNum; i++ {
		wait.Until(c.worker, time.Second*30, stopCh)
	}
	<-stopCh
}

func (c *appController) worker() {
	for {
		if !c.processNextEventKey() {
			fmt.Printf("handle event fail\n")
		}
	}
}

func (c *appController) processNextEventKey() bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)
	err := c.syncHandler(key.(string))
	if err != nil {
		c.handleErr(key.(string), err)
	}
	return true
}

func (c *appController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	app, err := c.appLister.Apps(namespace).Get(name)
	if errors.IsNotFound(err) {
		// delete deploy, service, ingress
		deploys, err := c.internalClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", controllerBy, name),
		})
		if err != nil {
			return err
		}

		for _, item := range deploys.Items {
			_ = c.internalClient.AppsV1().Deployments(namespace).Delete(context.TODO(), item.Name, metav1.DeleteOptions{})
		}
		services, err := c.internalClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", controllerBy, name),
		})
		if err != nil {
			return err
		}
		for _, item := range services.Items {
			_ = c.internalClient.CoreV1().Services(namespace).Delete(context.TODO(), item.Name, metav1.DeleteOptions{})
		}

		ingresses, err := c.internalClient.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", controllerBy, name),
		})
		if err != nil {
			return err
		}
		for _, item := range ingresses.Items {
			_ = c.internalClient.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), item.Name, metav1.DeleteOptions{})
		}
	}
	if err != nil {
		return err
	}

	deploy, err := c.deploymentLister.Deployments(namespace).Get(app.Spec.Deployment.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		// create deployment
		newDeploy := c.constructDeployment(app, 0)
		_, err := c.internalClient.AppsV1().Deployments(namespace).Create(context.TODO(), newDeploy, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		fmt.Println("create deployment success")
	} else if app.Spec.Deployment.Replicas != *deploy.Spec.Replicas {
		// update replicas
		updateDeploy := c.constructDeployment(app, app.Spec.Deployment.Replicas)
		_, err := c.internalClient.AppsV1().Deployments(namespace).Update(context.TODO(), updateDeploy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	svc, err := c.serviceLister.Services(namespace).Get(app.Spec.Service.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) && app.Spec.Service.Enabled {
		// create service
		newSvc := c.constructService(app)
		_, err := c.internalClient.CoreV1().Services(namespace).Create(context.TODO(), newSvc, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		fmt.Println("create service success")
	} else if !errors.IsNotFound(err) && !app.Spec.Service.Enabled {
		err := c.internalClient.CoreV1().Services(namespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		fmt.Println("delete service success")
	}

	ingress, err := c.ingressLister.Ingresses(namespace).Get(app.Spec.Ingress.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) && app.Spec.Service.Enabled && app.Spec.Ingress.Enabled {
		// create ingress
		newIng := c.constructIngress(app)
		_, err := c.internalClient.NetworkingV1().Ingresses(namespace).Create(context.TODO(), newIng, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		fmt.Println("create ingress success")
	} else if !errors.IsNotFound(err) {
		if !app.Spec.Service.Enabled || !app.Spec.Ingress.Enabled {
			// delete ingress
			err := c.internalClient.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), ingress.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
			fmt.Println("delete service success")
		}
	}
	return nil
}

func (c *appController) handleErr(key string, err error) {
	if c.queue.NumRequeues(key) >= 10 {
		c.queue.AddRateLimited(key)
		return
	}
	c.queue.Forget(key)
	runtime.HandleError(err)
}

func (c *appController) enqueue(key string) {
	c.queue.Add(key)
}

func (c *appController) deleteDeploymentEvent(obj interface{}) {
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	fmt.Printf("delete deploy event... %s\n", key)
	deploy := obj.(*deployapps.Deployment)
	ownerReference := metav1.GetControllerOf(deploy)
	if ownerReference.Kind != "App" {
		return
	}
	c.enqueue(deploy.Namespace + "/" + deploy.GetOwnerReferences()[0].Name)
}

func (c *appController) deleteSvcEvent(obj interface{}) {
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	fmt.Printf("delete service event... %s\n", key)
	svc := obj.(*core.Service)
	ownerReference := metav1.GetControllerOf(svc)
	if ownerReference.Kind != "App" {
		return
	}
	c.enqueue(svc.Namespace + "/" + svc.GetOwnerReferences()[0].Name)
}

func (c *appController) deleteIngressEvent(obj interface{}) {
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	fmt.Printf("delete ingress event... %s\n", key)
	ingress := obj.(*net.Ingress)
	ownerReference := metav1.GetControllerOf(ingress)
	if ownerReference.Kind != "App" {
		return
	}
	c.enqueue(ingress.Namespace + "/" + ingress.GetOwnerReferences()[0].Name)
}

func (c *appController) addAppEvent(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	fmt.Printf("add app event... %s\n", key)
	if err != nil {
		return
	}
	c.enqueue(key)
}

func (c *appController) updateAppEvent(oldObj interface{}, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	fmt.Printf("update app event... %s\n", key)
	if err != nil {
		return
	}
	c.enqueue(key)
}

func (c *appController) deleteAppEvent(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	fmt.Printf("delete app event... %s\n", key)
	if err != nil {
		return
	}
	c.enqueue(key)
}

func (c *appController) constructDeployment(app *appcontrollerv1.App, replicas int32) *deployapps.Deployment {
	labels := map[string]string{
		"app":        app.Name,
		"controller": app.Name,
	}
	if replicas <= 0 {
		replicas = app.Spec.Deployment.Replicas
	}
	return &deployapps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Deployment.Name,
			Namespace: app.Namespace,
			Labels: map[string]string{
				controllerBy: app.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, metav1.SchemeGroupVersion.WithKind("App")),
			},
		},
		Spec: deployapps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  app.Spec.Deployment.Name,
							Image: app.Spec.Deployment.Image,
						},
					},
				},
			},
		},
	}
}

func (c *appController) constructService(app *appcontrollerv1.App) *core.Service {
	labels := map[string]string{
		"app":        app.Name,
		"controller": app.Name,
	}
	return &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Service.Name,
			Namespace: app.Namespace,
			Labels: map[string]string{
				controllerBy: app.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, metav1.SchemeGroupVersion.WithKind("App")),
			},
		},
		Spec: core.ServiceSpec{
			Selector: labels,
			Ports: []core.ServicePort{
				{
					Protocol:   core.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 80},
				},
			},
		},
	}
}

func (c *appController) constructIngress(app *appcontrollerv1.App) *net.Ingress {
	ing := net.Ingress{}
	ing.Namespace = app.Namespace
	ing.Name = app.Spec.Ingress.Name
	ing.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(app, metav1.SchemeGroupVersion.WithKind("App"))}
	pathType := net.PathTypePrefix
	ingressClassName := "nginx"
	ing.Labels = map[string]string{
		controllerBy: app.Name,
	}
	ing.Spec = net.IngressSpec{
		IngressClassName: &ingressClassName,
		Rules: []net.IngressRule{
			{
				Host: "testing.com",
				IngressRuleValue: net.IngressRuleValue{
					HTTP: &net.HTTPIngressRuleValue{
						Paths: []net.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: net.IngressBackend{
									Service: &net.IngressServiceBackend{
										Name: app.Spec.Service.Name,
										Port: net.ServiceBackendPort{
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
	return &ing
}
