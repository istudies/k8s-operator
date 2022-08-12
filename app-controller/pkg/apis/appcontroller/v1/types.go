package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// generated command by: type-scaffold --kind App > pkg/apis/appcontroller/v1/types.go

type DeploymentObj struct {
	// deployment name
	Name string `json:"name"`
	// deployment image. e.g.: nginx:latest
	Image string `json:"image"`
	// deployment replications
	Replicas int32 `json:"replicas"`
}

type ServiceObj struct {
	// enabled service
	Enabled bool `json:"enabled"`
	// service name
	Name string `json:"name"`
}

type IngressObj struct {
	// enabled ingress
	Enabled bool `json:"enabled"`
	// ingress name
	Name string `json:"name"`
}

// AppSpec defines the desired state of App
type AppSpec struct {
	Deployment DeploymentObj `json:"deployment"`
	Service    ServiceObj    `json:"service"`
	Ingress    IngressObj    `json:"ingress"`
}

// AppStatus defines the observed state of App.
// It should always be reconstructable from the state of the cluster and/or outside world.
type AppStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS -- observed state of cluster
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// App is the Schema for the apps API
// +k8s:openapi-gen=true
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}
