package v1alpha1

import (
	"github.com/opendatahub-io/odh-platform-utilities/framework/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ api.PlatformObject = (*MaaSConfiguration)(nil)

// MaaSConfigurationSpec defines independently managed MaaS intent for an organization.
type MaaSConfigurationSpec struct {
	// organizationRef identifies the Organization served by this resource.
	OrganizationRef OrganizationReference `json:"organizationRef"`
	OIDC            *MaaSOIDC             `json:"oidc,omitempty"`
	TLS             *MaaSTLS              `json:"tls,omitempty"`
	Quotas          *MaaSQuotas           `json:"quotas,omitempty"`
}

type MaaSOIDC struct {
	// +kubebuilder:validation:MinLength=1
	ClientID string `json:"clientId"`
}

type MaaSTLS struct {
	CertificateRef MaaSCertificateRef `json:"certificateRef"`
}

type MaaSCertificateRef struct {
	// +kubebuilder:validation:MinLength=1
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}
type MaaSQuotas struct {
	MaxModels        int32 `json:"maxModels,omitempty"`
	MaxSubscriptions int32 `json:"maxSubscriptions,omitempty"`
	MaxAPIKeys       int32 `json:"maxApiKeys,omitempty"`
}

type MaaSConfigurationStatus struct {
	api.Status `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=maascfg
// +kubebuilder:printcolumn:name="Organization",type=string,JSONPath=`.spec.organizationRef.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type MaaSConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MaaSConfigurationSpec   `json:"spec"`
	Status            MaaSConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type MaaSConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaaSConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &MaaSConfiguration{}, &MaaSConfigurationList{})
		return nil
	})
}
func (m *MaaSConfiguration) GetStatus() *api.Status          { return &m.Status.Status }
func (m *MaaSConfiguration) GetConditions() []api.Condition  { return m.Status.GetConditions() }
func (m *MaaSConfiguration) SetConditions(c []api.Condition) { m.Status.SetConditions(c) }

var _ runtime.Object = (*MaaSConfiguration)(nil)
