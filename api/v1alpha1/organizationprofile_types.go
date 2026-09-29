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
var _ api.PlatformObject = (*OrganizationProfile)(nil)

// OrganizationProfileSpec defines the desired state of OrganizationProfile.
// It holds an organization's self-managed configuration. It is auto-created 1:1
// with an Organization with restrictive defaults, then managed by organization admins.
type OrganizationProfileSpec struct {
	// organizationRef identifies the Organization this profile configures. It is
	// set at creation and immutable (enforced by webhook).
	OrganizationRef OrganizationReference `json:"organizationRef"`

	// admins are subjects with administrative authority over this organization.
	// Ancestor admins additionally inherit authority via the hierarchy.
	// +optional
	Admins []Subject `json:"admins,omitempty"`

	// defaults are applied to OrganizationProjects created in this organization.
	// +optional
	Defaults ProjectDefaults `json:"defaults,omitempty"`

	// platform contains configuration shared by platform capabilities.
	// +optional
	Platform *OrganizationPlatformConfig `json:"platform,omitempty"`
}

// OrganizationReference identifies an Organization by name.
type OrganizationReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// OrganizationPlatformConfig contains settings shared by platform capabilities.
type OrganizationPlatformConfig struct {
	// oidc contains the shared issuer. Client IDs remain capability-specific.
	// +optional
	OIDC *OIDCConfig `json:"oidc,omitempty"`

	// ingressGatewayRef identifies the shared Kubernetes Gateway API Gateway.
	// +optional
	IngressGatewayRef *GatewayReference `json:"ingressGatewayRef,omitempty"`
}

type OIDCConfig struct {
	// +kubebuilder:validation:MinLength=1
	IssuerURL string `json:"issuerUrl"`
}

// GatewayReference identifies a Kubernetes Gateway API object.
type GatewayReference struct {
	// +kubebuilder:validation:MinLength=1
	Group string `json:"group"`
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`
	// +kubebuilder:validation:MinLength=1
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// ProjectDefaults are the restrictive defaults an organization applies to its projects.
type ProjectDefaults struct {
	// networkIsolation is the default isolation preset for new projects.
	// +kubebuilder:validation:Enum=none;tenant;strict
	// +kubebuilder:default=tenant
	// +optional
	NetworkIsolation string `json:"networkIsolation,omitempty"`

	// maxProjects caps how many OrganizationProjects this organization may own. Zero (the
	// restrictive default) means no projects until an admin raises it.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// +optional
	MaxProjects int32 `json:"maxProjects,omitempty"`
}

// OrganizationProfileStatus defines the observed state of OrganizationProfile.
type OrganizationProfileStatus struct {
	// Embed common status for PlatformObject compliance
	api.Status `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=orgprof
// +kubebuilder:printcolumn:name="Organization",type=string,JSONPath=`.spec.organizationRef.name`
// +kubebuilder:printcolumn:name="MaxProjects",type=integer,JSONPath=`.spec.defaults.maxProjects`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OrganizationProfile is the Schema for the organizationprofiles API.
type OrganizationProfile struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of OrganizationProfile
	// +required
	Spec OrganizationProfileSpec `json:"spec"`

	// status defines the observed state of OrganizationProfile
	// +optional
	Status OrganizationProfileStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OrganizationProfileList contains a list of OrganizationProfile.
type OrganizationProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OrganizationProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OrganizationProfile{}, &OrganizationProfileList{})
		return nil
	})
}

func (p *OrganizationProfile) GetStatus() *api.Status {
	return &p.Status.Status
}

func (p *OrganizationProfile) GetConditions() []api.Condition {
	return p.Status.GetConditions()
}

func (p *OrganizationProfile) SetConditions(conditions []api.Condition) {
	p.Status.SetConditions(conditions)
}
