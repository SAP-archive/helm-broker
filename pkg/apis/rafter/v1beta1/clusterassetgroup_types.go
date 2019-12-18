package v1beta1

//Copied from https://github.com/kyma-project/rafter/tree/9c356a443bda8b324ad4cefbf16cf449985c880a/pkg/apis/rafter/v1beta1
//Because of conflicts of the dependencies (especially api-machinery)
//Could be removed after update helm-brokers api-machinery version to >=0.2.0

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterAssetGroupSpec defines the desired state of ClusterAssetGroup
type ClusterAssetGroupSpec struct {
	CommonAssetGroupSpec `json:",inline"`
}

// ClusterAssetGroupStatus defines the observed state of ClusterAssetGroup
type ClusterAssetGroupStatus struct {
	CommonAssetGroupStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClusterAssetGroup is the Schema for the clusterassetgroups API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ClusterAssetGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterAssetGroupSpec   `json:"spec,omitempty"`
	Status ClusterAssetGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterAssetGroupList contains a list of ClusterAssetGroup
type ClusterAssetGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterAssetGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterAssetGroup{}, &ClusterAssetGroupList{})
}
