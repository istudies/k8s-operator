package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	Schema       = runtime.NewScheme()
	GroupVersion = schema.GroupVersion{
		Group:   "ctl.mycrds.com",
		Version: "v1",
	}
	Codec = serializer.NewCodecFactory(Schema)
)

func init() {
	// register custom resource
	Schema.AddKnownTypes(GroupVersion, &MyCRD{}, &MyCRDList{})
}
