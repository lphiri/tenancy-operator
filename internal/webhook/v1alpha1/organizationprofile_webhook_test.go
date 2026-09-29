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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("OrganizationProfile Webhook", func() {
	var (
		obj       *tenancyv1alpha1.OrganizationProfile
		oldObj    *tenancyv1alpha1.OrganizationProfile
		validator OrganizationProfileCustomValidator
	)

	BeforeEach(func() {
		obj = &tenancyv1alpha1.OrganizationProfile{}
		oldObj = &tenancyv1alpha1.OrganizationProfile{}
		validator = OrganizationProfileCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating or updating OrganizationProfile under Validating Webhook", func() {
		It("rejects a name and organizationRef mismatch", func() {
			obj.Name = "different"
			obj.Spec.OrganizationRef.Name = "organization"

			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(MatchError(ContainSubstring("must equal spec.organizationRef.name")))
		})

		It("rejects organizationRef changes", func() {
			oldObj.Spec.OrganizationRef.Name = testOldOrganizationName
			obj.Spec.OrganizationRef.Name = testNewOrganizationName

			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(MatchError(ContainSubstring("spec.organizationRef.name is immutable")))
		})
	})

})
