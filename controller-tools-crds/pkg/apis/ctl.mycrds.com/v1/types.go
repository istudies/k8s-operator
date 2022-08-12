package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// by: type-scaffold --kind MyCRD > pkg/apis/ctl.mycrds.com/v1/types.go

// MyCRDSpec defines the desired state of MyCRD
type MyCRDSpec struct {
	// special crd name
	Name string `json:"name"`
	// special crd replication
	Replicas int32 `json:"replicas"`
}

// MyCRDStatus defines the observed state of MyCRD.
// It should always be reconstructable from the state of the cluster and/or outside world.
type MyCRDStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS -- observed state of cluster
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MyCRD is the Schema for the mycrds API
// +k8s:openapi-gen=true
type MyCRD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyCRDSpec   `json:"spec,omitempty"`
	Status MyCRDStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MyCRDList contains a list of MyCRD
type MyCRDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyCRD `json:"items"`
}
