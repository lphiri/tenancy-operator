package main

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSchemeRegistersCustomResourceDefinitions(t *testing.T) {
	gvk := schema.GroupVersionKind{
		Group:   apiextensionsv1.SchemeGroupVersion.Group,
		Version: apiextensionsv1.SchemeGroupVersion.Version,
		Kind:    "CustomResourceDefinition",
	}

	obj, err := scheme.New(gvk)
	if err != nil {
		t.Fatalf("create %s from scheme: %v", gvk, err)
	}
	if _, ok := obj.(*apiextensionsv1.CustomResourceDefinition); !ok {
		t.Fatalf("scheme returned %T for %s", obj, gvk)
	}
}
