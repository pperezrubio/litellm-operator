/*
Copyright 2026 bitkaio LLC.

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

package controller

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	litellmv1alpha1 "github.com/PalenaAI/litellm-operator/api/v1alpha1"
	"github.com/PalenaAI/litellm-operator/internal/litellm"
)

var _ = Describe("LiteLLMModel Controller", func() {
	Describe("model adoption helpers", func() {
		It("filters models by name and picks a deterministic order", func() {
			models := []litellm.ModelInfoResponse{
				{ModelID: "model-b", ModelName: "gpt-4"},
				{ModelID: "ignored", ModelName: "gpt-3.5"},
				{ModelID: "model-a", ModelName: "gpt-4"},
				{ModelID: "", ModelName: "gpt-4"},
			}

			matches := modelsByName(models, "gpt-4")

			Expect(matches).To(Equal([]litellm.ModelInfoResponse{
				{ModelID: "model-a", ModelName: "gpt-4"},
				{ModelID: "model-b", ModelName: "gpt-4"},
			}))
		})

		It("deletes duplicate model IDs while keeping the desired model", func() {
			mockClient := litellm.NewMockClient()
			deleted := []string{}
			mockClient.MockModels.DeleteFunc = func(ctx context.Context, modelID string) error {
				deleted = append(deleted, modelID)
				return nil
			}

			err := deleteDuplicateModels(context.Background(), mockClient, "model-a", []litellm.ModelInfoResponse{
				{ModelID: "model-a", ModelName: "gpt-4"},
				{ModelID: "model-b", ModelName: "gpt-4"},
				{ModelID: "", ModelName: "gpt-4"},
				{ModelID: "model-c", ModelName: "gpt-4"},
			})

			Expect(err).NotTo(HaveOccurred())
			slices.Sort(deleted)
			Expect(deleted).To(Equal([]string{"model-b", "model-c"}))
		})
	})

	Context("When reconciling a resource", func() {
		const resourceName = "test-model"
		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"}

		BeforeEach(func() {
			model := &litellmv1alpha1.LiteLLMModel{}
			err := k8sClient.Get(ctx, typeNamespacedName, model)
			if err != nil && errors.IsNotFound(err) {
				resource := &litellmv1alpha1.LiteLLMModel{
					ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: "default"},
					Spec: litellmv1alpha1.LiteLLMModelSpec{
						InstanceRef: litellmv1alpha1.InstanceRef{Name: "test-instance"},
						ModelName:   "gpt-4",
						LiteLLMParams: litellmv1alpha1.LiteLLMModelParams{
							Model: "openai/gpt-4",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &litellmv1alpha1.LiteLLMModel{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			controllerReconciler := &LiteLLMModelReconciler{
				Client:               k8sClient,
				Scheme:               k8sClient.Scheme(),
				LiteLLMClientFactory: func(endpoint, masterKey string) litellm.Client { return litellm.NewMockClient() },
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			// Expected to requeue since the instance doesn't exist
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
