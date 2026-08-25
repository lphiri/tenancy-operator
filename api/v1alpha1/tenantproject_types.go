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

// TenantProjectSpec defines the desired state of TenantProject.
// A TenantProject provisions one Namespace with RBAC and NetworkPolicies.
type TenantProjectSpec struct {
	// tenant is the name of the owning PlatformTenant. Immutable (webhook).
	// +kubebuilder:validation:MinLength=1
	Tenant string `json:"tenant"`

	// displayName is a human-friendly name for the project.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// users are granted access to the project namespace via RoleBindings.
	// +optional
	Users []ProjectUser `json:"users,omitempty"`

	// networkIsolation is the isolation preset for the project namespace.
	// Empty means inherit the tenant default.
	// +kubebuilder:validation:Enum=none;tenant;strict
	// +optional
	NetworkIsolation string `json:"networkIsolation,omitempty"`
}

// ProjectUser grants a subject a role within the project namespace.
type ProjectUser struct {
	// kind is the RBAC subject kind.
	// +kubebuilder:validation:Enum=User;Group;ServiceAccount
	Kind string `json:"kind"`

	// name is the subject name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// namespace is required only when kind is ServiceAccount.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// role is the access level granted, mapped to the built-in edit/view roles.
	// +kubebuilder:validation:Enum=edit;view
	Role string `json:"role"`
}

// TenantProjectStatus defines the observed state of TenantProject.
type TenantProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the TenantProject resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// namespace is the name of the provisioned project Namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=tproj
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenant`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TenantProject is the Schema for the tenantprojects API
type TenantProject struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of TenantProject
	// +required
	Spec TenantProjectSpec `json:"spec"`

	// status defines the observed state of TenantProject
	// +optional
	Status TenantProjectStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TenantProjectList contains a list of TenantProject
type TenantProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TenantProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &TenantProject{}, &TenantProjectList{})
		return nil
	})
}
