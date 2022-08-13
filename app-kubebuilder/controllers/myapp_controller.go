/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	mykbmev1 "github.com/istudies/k8s-operator/app-kubebuilder/api/v1"
	"github.com/istudies/k8s-operator/app-kubebuilder/controllers/utils"
	deployapps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MyAppReconciler reconciles a MyApp object
type MyAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mykb.me.my.domain,resources=myapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mykb.me.my.domain,resources=myapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mykb.me.my.domain,resources=myapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the MyApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *MyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	//logger.Info("current event: " + req.String())
	// get app crd
	app := &mykbmev1.MyApp{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// deployments
	deployment, err := utils.NewDeployment(app)
	if err != nil {
		return ctrl.Result{}, err
	}
	// set deployment owner reference
	if err = controllerutil.SetControllerReference(app, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	d := &deployapps.Deployment{}
	if err = r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, d); err != nil {
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, deployment); err != nil {
				logger.Error(err, "create deploy failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if err = r.Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
	}

	// service
	service, err := utils.NewService(app)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err = controllerutil.SetControllerReference(app, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	s := &corev1.Service{}
	if err = r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, s); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnabledService {
			if err = r.Create(ctx, service); err != nil {
				logger.Error(err, "create service failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if app.Spec.EnabledService {
			if err = r.Update(ctx, service); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err = r.Delete(ctx, s); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// ingress
	ingress, err := utils.NewIngress(app)
	if err != nil {
		return ctrl.Result{}, nil
	}
	if err = controllerutil.SetControllerReference(app, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	i := &netv1.Ingress{}
	if err = r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, i); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnabledIngress && app.Spec.EnabledService {
			if err = r.Create(ctx, ingress); err != nil {
				logger.Error(err, "create service failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if app.Spec.EnabledService && app.Spec.EnabledIngress {
			if err = r.Update(ctx, ingress); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err = r.Delete(ctx, i); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mykbmev1.MyApp{}).
		// add built-in resource listen
		Owns(&deployapps.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}
