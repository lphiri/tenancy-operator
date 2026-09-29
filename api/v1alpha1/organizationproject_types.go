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
	"github.com/opendatahub-io/odh-platform-utilities/framework/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Check that it implements common.PlatformObject.
var _ api.PlatformObject = (*OrganizationProject)(nil)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OrganizationProjectSpec defines the desired state of OrganizationProject.
// An OrganizationProject provisions one Namespace with RBAC and NetworkPolicies.
type OrganizationProjectSpec struct {
	// organizationRef identifies the owning Organization. Immutable (webhook).
	OrganizationRef OrganizationReference `json:"organizationRef"`

	// users are granted access to the project namespace via RoleBindings.
	// +optional
	Users []ProjectUser `json:"users,omitempty"`

	// networkIsolation is the isolation preset for the project namespace.
	// Empty means inherit the organization default.
	// +kubebuilder:validation:Enum=none;tenant;strict
	// +optional
	NetworkIsolation string `json:"networkIsolation,omitempty"`

	// networkGrants contains additive cross-project network access requests.
	// +optional
	NetworkGrants []NetworkGrant `json:"networkGrants,omitempty"`
}

// NetworkGrant describes an additive network access request.
type NetworkGrant struct {
	// to is the target OrganizationProject reference, for example org/project.
	// +kubebuilder:validation:MinLength=1
	To string `json:"to"`
	// direction is the requested traffic direction.
	// +kubebuilder:validation:Enum=ingress;egress
	Direction string `json:"direction"`
	// ports limits the grant to the listed ports. An empty list means all ports.
	// +kubebuilder:validation:items:Minimum=1
	// +kubebuilder:validation:items:Maximum=65535
	// +optional
	Ports []int32 `json:"ports,omitempty"`
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

// OrganizationProjectStatus defines the observed state of OrganizationProject.
type OrganizationProjectStatus struct {
	// Embed common status for PlatformObject compliance
	api.Status `json:",inline"`

	// namespace is the name of the provisioned project Namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=orgproj
// +kubebuilder:printcolumn:name="Organization",type=string,JSONPath=`.spec.organizationRef.name`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OrganizationProject is the Schema for the organizationprojects API.
type OrganizationProject struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of OrganizationProject
	// +required
	Spec OrganizationProjectSpec `json:"spec"`

	// status defines the observed state of OrganizationProject
	// +optional
	Status OrganizationProjectStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OrganizationProjectList contains a list of OrganizationProject.
type OrganizationProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OrganizationProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OrganizationProject{}, &OrganizationProjectList{})
		return nil
	})
}

func (p *OrganizationProject) GetStatus() *api.Status {
	return &p.Status.Status
}

func (p *OrganizationProject) GetConditions() []api.Condition {
	return p.Status.GetConditions()
}

func (p *OrganizationProject) SetConditions(conditions []api.Condition) {
	p.Status.SetConditions(conditions)
}
