package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "make generates" to regenerate files after modifying those structs

// +kubebuilder:object:root=true

// AddonsConfiguration is the Schema for the addonsconfigurations API
//
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:categories=all;addons
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type AddonsConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonsConfigurationSpec   `json:"spec,omitempty"`
	Status AddonsConfigurationStatus `json:"status,omitempty"`
}

// AddonsConfigurationSpec defines the desired state of AddonsConfiguration
type AddonsConfigurationSpec struct {
	CommonAddonsConfigurationSpec `json:",inline"`
}

// AddonsConfigurationStatus defines the observed state of AddonsConfiguration
type AddonsConfigurationStatus struct {
	CommonAddonsConfigurationStatus `json:",inline"`
}

// AddonsConfigurationList contains a list of AddonsConfiguration
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonsConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddonsConfiguration `json:"items"`
}

// +kubebuilder:object:root=true

// ClusterAddonsConfiguration is the Schema for the addonsconfigurations API
//
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:categories=all,addons
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ClusterAddonsConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterAddonsConfigurationSpec   `json:"spec,omitempty"`
	Status ClusterAddonsConfigurationStatus `json:"status,omitempty"`
}

// ClusterAddonsConfigurationSpec defines the desired state of ClusterAddonsConfiguration
type ClusterAddonsConfigurationSpec struct {
	CommonAddonsConfigurationSpec `json:",inline"`
}

// ClusterAddonsConfigurationStatus defines the observed state of ClusterAddonsConfiguration
type ClusterAddonsConfigurationStatus struct {
	CommonAddonsConfigurationStatus `json:",inline"`
}

// ClusterAddonsConfigurationList contains a list of ClusterAddonsConfiguration
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterAddonsConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterAddonsConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&AddonsConfiguration{}, &AddonsConfigurationList{},
		&ClusterAddonsConfiguration{}, &ClusterAddonsConfigurationList{},
	)
}
