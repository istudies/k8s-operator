/*******************************************************************************
 * @File: resource_parse.go
 * @Description:
 * @Author: jiangxunyu
 * @Version: 1.0.0
 * @Date: 2022/8/13 17:51
*******************************************************************************/

package utils

import (
	"bytes"
	"text/template"

	corev1 "k8s.io/api/core/v1"

	netv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/util/yaml"

	appsv1 "k8s.io/api/apps/v1"

	v1 "github.com/istudies/k8s-operator/app-kubebuilder/api/v1"
)

func parseTemplates(templateName string, app *v1.MyApp) ([]byte, error) {
	tmpl, err := template.ParseFiles("controllers/templates/" + templateName + ".yaml")
	if err != nil {
		return nil, err
	}
	b := new(bytes.Buffer)
	err = tmpl.Execute(b, app)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func NewDeployment(app *v1.MyApp) (*appsv1.Deployment, error) {
	b, err := parseTemplates("deployment", app)
	if err != nil {
		return nil, err
	}
	res := &appsv1.Deployment{}
	if err = yaml.Unmarshal(b, res); err != nil {
		return nil, err
	}
	return res, nil
}

func NewService(app *v1.MyApp) (*corev1.Service, error) {
	b, err := parseTemplates("service", app)
	if err != nil {
		return nil, err
	}
	res := &corev1.Service{}
	if err = yaml.Unmarshal(b, res); err != nil {
		return nil, err
	}
	return res, nil
}

func NewIngress(app *v1.MyApp) (*netv1.Ingress, error) {
	b, err := parseTemplates("ingress", app)
	if err != nil {
		return nil, err
	}
	res := &netv1.Ingress{}
	if err = yaml.Unmarshal(b, res); err != nil {
		return nil, err
	}
	return res, nil
}
