package v1beta1

//Copied from https://github.com/kyma-project/rafter/tree/9c356a443bda8b324ad4cefbf16cf449985c880a/pkg/apis/rafter/v1beta1
//Because of conflicts of the dependencies (especially api-machinery)
//Could be removed after update helm-brokers api-machinery version to >=0.2.0

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BucketSpec defines the desired state of Bucket
type BucketSpec struct {
	CommonBucketSpec `json:",inline"`
}

// BucketStatus defines the observed state of Bucket
type BucketStatus struct {
	CommonBucketStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// Bucket is the Schema for the buckets API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec,omitempty"`
	Status BucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket
type BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
}
