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

// TenantProfileSpec defines the desired state of TenantProfile.
// It holds a tenant's self-managed configuration. It is auto-created 1:1 with a
// PlatformTenant with restrictive defaults, then managed by the tenant admins.
type TenantProfileSpec struct {
	// tenant is the name of the PlatformTenant this profile configures. It is
	// set at creation and immutable (enforced by webhook).
	// +kubebuilder:validation:MinLength=1
	Tenant string `json:"tenant"`

	// admins are subjects with administrative authority over this tenant.
	// Ancestor admins additionally inherit authority via the hierarchy.
	// +optional
	Admins []Subject `json:"admins,omitempty"`

	// defaults are applied to TenantProjects created in this tenant.
	// +optional
	Defaults ProjectDefaults `json:"defaults,omitempty"`
}

// ProjectDefaults are the restrictive defaults a tenant applies to its projects.
type ProjectDefaults struct {
	// networkIsolation is the default isolation preset for new projects.
	// +kubebuilder:validation:Enum=none;tenant;strict
	// +kubebuilder:default=tenant
	// +optional
	NetworkIsolation string `json:"networkIsolation,omitempty"`

	// maxProjects caps how many TenantProjects this tenant may own. Zero (the
	// restrictive default) means no projects until an admin raises it.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// +optional
	MaxProjects int32 `json:"maxProjects,omitempty"`
}

// TenantProfileStatus defines the observed state of TenantProfile.
type TenantProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the TenantProfile resource.
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
// +kubebuilder:resource:scope=Cluster,shortName=tprof
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenant`
// +kubebuilder:printcolumn:name="MaxProjects",type=integer,JSONPath=`.spec.defaults.maxProjects`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TenantProfile is the Schema for the tenantprofiles API
type TenantProfile struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of TenantProfile
	// +required
	Spec TenantProfileSpec `json:"spec"`

	// status defines the observed state of TenantProfile
	// +optional
	Status TenantProfileStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TenantProfileList contains a list of TenantProfile
type TenantProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TenantProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &TenantProfile{}, &TenantProfileList{})
		return nil
	})
}
