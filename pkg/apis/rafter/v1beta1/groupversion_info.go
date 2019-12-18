// Package v1beta1 contains API Schema definitions for the rafter v1beta1 API group
// +kubebuilder:object:generate=true
// +groupName=rafter.kyma-project.io
package v1beta1

//Copied from https://github.com/kyma-project/rafter/tree/9c356a443bda8b324ad4cefbf16cf449985c880a/pkg/apis/rafter/v1beta1
//Because of conflicts of the dependencies (especially api-machinery)
//Could be removed after update helm-brokers api-machinery version to >=0.2.0

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "rafter.kyma-project.io", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
