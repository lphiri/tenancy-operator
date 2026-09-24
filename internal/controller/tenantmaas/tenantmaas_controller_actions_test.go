package tenantmaas

import (
	"github.com/onsi/gomega"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRenderAITenant(t *testing.T) {
	g := gomega.NewWithT(t)
	obj := &unstructured.Unstructured{Object: map[string]any{}}
	spec := &tenancyv1alpha1.TenantMaaSSpec{
		OIDC:    &tenancyv1alpha1.MaaSOIDC{IssuerURL: "https://issuer.example", ClientID: "client"},
		Gateway: &tenancyv1alpha1.MaaSGateway{Name: "gateway"},
		TLS:     &tenancyv1alpha1.MaaSTLS{CertificateRef: tenancyv1alpha1.MaaSCertificateRef{Name: "tls", Namespace: "ai-tenants"}},
		Quotas:  &tenancyv1alpha1.MaaSQuotas{MaxModels: 2, MaxSubscriptions: 3, MaxAPIKeys: 4},
	}
	g.Expect(render(obj, spec)).To(gomega.Succeed())
	g.Expect(obj.Object["spec"]).To(gomega.Equal(map[string]any{
		"oidc":           map[string]any{"issuerUrl": "https://issuer.example", "clientId": "client"},
		"gateway":        map[string]any{"name": "gateway"},
		"tls":            map[string]any{"certificateRef": map[string]any{"name": "tls", "namespace": "ai-tenants"}},
		"resourceQuotas": map[string]any{"maxModels": int64(2), "maxSubscriptions": int64(3), "maxApiKeys": int64(4)},
	}))
}
