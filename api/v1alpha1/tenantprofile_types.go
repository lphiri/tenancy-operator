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
var _ api.PlatformObject = (*TenantProfile)(nil)

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

	// Services contains optional platform services provisioned for this tenant.
	// +optional
	Services TenantServices `json:"services,omitempty"`
}

// TenantServices defines optional delegated platform services.
type TenantServices struct {
	// MaaS enables delegated Models-as-a-Service tenant provisioning. MaaS is
	// currently supported only for root tenants.
	// +optional
	MaaS *MaaSService `json:"maas,omitempty"`
}

// MaaSService is the desired configuration for a provisioned AITenant.
type MaaSService struct {
	// Enabled controls whether the tenancy controller owns an AITenant.
	Enabled bool `json:"enabled,omitempty"`
	// OIDC configures the identity provider used by the AI Gateway.
	// +optional
	OIDC *MaaSOIDC `json:"oidc,omitempty"`
	// Gateway references a network-admin-provisioned Gateway.
	// +optional
	Gateway *MaaSGateway `json:"gateway,omitempty"`
	// TLS references the certificate used by the tenant gateway.
	// +optional
	TLS *MaaSTLS `json:"tls,omitempty"`
	// Quotas limits MaaS control-plane objects, not compute resources.
	// +optional
	Quotas *MaaSQuotas `json:"quotas,omitempty"`
}

type MaaSOIDC struct {
	IssuerURL string `json:"issuerUrl"`
	ClientID  string `json:"clientId"`
}

type MaaSGateway struct {
	Name string `json:"name"`
}

type MaaSTLS struct {
	CertificateRef MaaSCertificateRef `json:"certificateRef"`
}

type MaaSCertificateRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type MaaSQuotas struct {
	MaxModels        int32 `json:"maxModels,omitempty"`
	MaxSubscriptions int32 `json:"maxSubscriptions,omitempty"`
	MaxAPIKeys       int32 `json:"maxApiKeys,omitempty"`
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
	// Embed common status for PlatformObject compliance
	api.Status `json:",inline"`

	// MaaS reports the status of the owned AITenant, when enabled.
	// +optional
	MaaS MaaSStatus `json:"maas,omitempty"`
}

type MaaSStatus struct {
	// +optional
	AITenant string `json:"aiTenant,omitempty"`
	// +optional
	Ready bool `json:"ready,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
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

func (tprof *TenantProfile) GetStatus() *api.Status {
	return &tprof.Status.Status
}

func (tprof *TenantProfile) GetConditions() []api.Condition {
	return tprof.Status.GetConditions()
}

func (tprof *TenantProfile) SetConditions(conditions []api.Condition) {
	tprof.Status.SetConditions(conditions)
}
