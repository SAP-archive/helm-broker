package v1beta1

//Copied from https://github.com/kyma-project/rafter/tree/9c356a443bda8b324ad4cefbf16cf449985c880a/pkg/apis/rafter/v1beta1
//Because of conflicts of the dependencies (especially api-machinery)
//Could be removed after update helm-brokers api-machinery version to >=0.2.0

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterBucketSpec defines the desired state of ClusterBucket
type ClusterBucketSpec struct {
	CommonBucketSpec `json:",inline"`
}

// ClusterBucketStatus defines the observed state of ClusterBucket
type ClusterBucketStatus struct {
	CommonBucketStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterBucket is the Schema for the clusterbuckets API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ClusterBucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterBucketSpec   `json:"spec,omitempty"`
	Status ClusterBucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterBucketList contains a list of ClusterBucket
type ClusterBucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterBucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterBucket{}, &ClusterBucketList{})
}
