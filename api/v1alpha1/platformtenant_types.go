/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PlatformTenantSpec defines the desired state of PlatformTenant.
// It carries hierarchy and existence only: no capacity, quota, or hardware.
type PlatformTenantSpec struct {
	// displayName is a human-friendly name for the tenant.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// parent is the name of the parent PlatformTenant. An empty parent means
	// this is a root tenant. Immutable after creation (enforced by webhook).
	// +optional
	Parent string `json:"parent,omitempty"`
}

// PlatformTenantStatus defines the observed state of PlatformTenant.
type PlatformTenantStatus struct {
	// root is the name of the root tenant at the top of this tenant's lineage.
	// +optional
	Root string `json:"root,omitempty"`

	// conditions represent the current state of the PlatformTenant resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=pt
// +kubebuilder:printcolumn:name="Parent",type=string,JSONPath=`.spec.parent`
// +kubebuilder:printcolumn:name="Root",type=string,JSONPath=`.status.root`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PlatformTenant is the Schema for the platformtenants API
type PlatformTenant struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PlatformTenant
	// +required
	Spec PlatformTenantSpec `json:"spec"`

	// status defines the observed state of PlatformTenant
	// +optional
	Status PlatformTenantStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PlatformTenantList contains a list of PlatformTenant
type PlatformTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PlatformTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &PlatformTenant{}, &PlatformTenantList{})
		return nil
	})
}
