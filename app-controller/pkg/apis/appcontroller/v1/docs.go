// +k8s:deepcopy-gen=package
// +groupName=appcontroller.me
package v1

// code-generator:  deep copy, lister, informer, client set
// controller-tool: type scaffold, crd, deep copy

// gen crd yaml by: controller-gen crd paths=./... output:crd:dir=./manifest output:stdout
// gen deep copy by: controller-gen object paths=./pkg/apis/appcontroller/v1/types.go

/*
See:

# outputting crds to /tmp/crds and everything else to stdout
controller-gen rbac:roleName=<role name> crd paths=./apis/... output:crd:dir=/tmp/crds output:stdout

# Generate deepcopy/runtime.Object implementations for a particular file
controller-gen object paths=./apis/v1beta1/some_types.go
*/
