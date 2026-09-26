package maasconfiguration

import (
	"testing"

	"github.com/onsi/gomega"
	organizationv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	testTLSName     = "tls"
	testGatewayName = "gateway"
)

func TestRenderAITenant(t *testing.T) {
	g := gomega.NewWithT(t)
	obj := &unstructured.Unstructured{Object: map[string]any{}}
	spec := &organizationv1alpha1.MaaSConfigurationSpec{
		OIDC:   &organizationv1alpha1.MaaSOIDC{ClientID: "client"},
		TLS:    &organizationv1alpha1.MaaSTLS{CertificateRef: organizationv1alpha1.MaaSCertificateRef{Name: testTLSName, Namespace: "ai-tenants"}},
		Quotas: &organizationv1alpha1.MaaSQuotas{MaxModels: 2, MaxSubscriptions: 3, MaxAPIKeys: 4},
	}
	profile := &organizationv1alpha1.OrganizationProfile{Spec: organizationv1alpha1.OrganizationProfileSpec{
		Platform: &organizationv1alpha1.OrganizationPlatformConfig{
			OIDC:              &organizationv1alpha1.OIDCConfig{IssuerURL: "https://issuer.example"},
			IngressGatewayRef: &organizationv1alpha1.GatewayReference{Name: testGatewayName},
		},
	}}
	g.Expect(render(obj, spec, profile)).To(gomega.Succeed())
	g.Expect(obj.Object["spec"]).To(gomega.Equal(map[string]any{
		"oidc":           map[string]any{"issuerUrl": "https://issuer.example", "clientId": "client"},
		gatewayField:     map[string]any{nestedNameField: testGatewayName},
		tlsField:         map[string]any{"certificateRef": map[string]any{nestedNameField: testTLSName, "namespace": "ai-tenants"}},
		"resourceQuotas": map[string]any{"maxModels": int64(2), "maxSubscriptions": int64(3), "maxApiKeys": int64(4)},
	}))
}
