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

// Check that it implements api.PlatformObject.
var _ api.PlatformObject = (*Organization)(nil)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OrganizationSpec defines the desired state of an Organization.
// It carries hierarchy and existence only: no capacity, quota, or hardware.
type OrganizationSpec struct {
	// parent is the name of the parent Organization. An empty parent means
	// this is a root organization. Immutable after creation (enforced by webhook).
	// +optional
	Parent string `json:"parent,omitempty"`
}

// OrganizationStatus defines the observed state of an Organization.
type OrganizationStatus struct {
	// Embed common status for PlatformObject compliance
	api.Status `json:",inline"`

	// root is the name of the root organization at the top of this organization's lineage.
	// +optional
	Root string `json:"root,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=org
// +kubebuilder:printcolumn:name="Parent",type=string,JSONPath=`.spec.parent`
// +kubebuilder:printcolumn:name="Root",type=string,JSONPath=`.status.root`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Organization is the Schema for the organizations API.
type Organization struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Organization
	// +required
	Spec OrganizationSpec `json:"spec"`

	// status defines the observed state of Organization
	// +optional
	Status OrganizationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OrganizationList contains a list of Organization.
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Organization `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Organization{}, &OrganizationList{})
		return nil
	})
}

func (o *Organization) GetStatus() *api.Status {
	return &o.Status.Status
}

func (o *Organization) GetConditions() []api.Condition {
	return o.Status.GetConditions()
}

func (o *Organization) SetConditions(conditions []api.Condition) {
	o.Status.SetConditions(conditions)
}
