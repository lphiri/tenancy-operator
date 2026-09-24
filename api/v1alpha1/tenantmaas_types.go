package v1alpha1

import (
	"github.com/opendatahub-io/odh-platform-utilities/framework/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ api.PlatformObject = (*TenantMaaS)(nil)

// TenantMaaSSpec defines an independently managed MaaS service for a tenant.
type TenantMaaSSpec struct {
	// tenantRef identifies the PlatformTenant served by this resource.
	TenantRef TenantReference `json:"tenantRef"`
	OIDC      *MaaSOIDC       `json:"oidc,omitempty"`
	Gateway   *MaaSGateway    `json:"gateway,omitempty"`
	TLS       *MaaSTLS        `json:"tls,omitempty"`
	Quotas    *MaaSQuotas     `json:"quotas,omitempty"`
}

type TenantReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
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

type TenantMaaSStatus struct {
	api.Status `json:",inline"`
	AITenant   string `json:"aiTenant,omitempty"`
	Ready      bool   `json:"ready,omitempty"`
	Message    string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=tmaas
type TenantMaaS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TenantMaaSSpec   `json:"spec"`
	Status            TenantMaaSStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type TenantMaaSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantMaaS `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &TenantMaaS{}, &TenantMaaSList{})
		return nil
	})
}
func (t *TenantMaaS) GetStatus() *api.Status          { return &t.Status.Status }
func (t *TenantMaaS) GetConditions() []api.Condition  { return t.Status.GetConditions() }
func (t *TenantMaaS) SetConditions(c []api.Condition) { t.Status.SetConditions(c) }

var _ runtime.Object = (*TenantMaaS)(nil)
