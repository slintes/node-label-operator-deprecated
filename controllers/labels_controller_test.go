package controllers

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/slintes/node-label-operator/api/v1beta1"
)

var _ = Describe("Labels controller", func() {

	Context("When creating a Labels CR", func() {
		It("Should add labels to matching node", func() {
			By("Creating a node")
			node := getNode(nodeNameMatching)
			Expect(k8sClient.Create(ctx, node)).Should(Succeed(), "node should have been created")

			By("Creating a Labels CR")
			labels := &v1beta1.Labels{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Labels",
					APIVersion: "labels.slintes.net/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-labels-",
					Namespace:    "default",
				},
				Spec: v1beta1.LabelsSpec{
					Rules: []v1beta1.Rule{
						{
							NodeNamePatterns: []string{nodeNamePattern},
							Labels:           []string{label},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, labels)).Should(Succeed(), "labels should have been created")

			By("checking if label was set")
			key := types.NamespacedName{
				Name: node.Name,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), key, node)
				Expect(err).ToNot(HaveOccurred())
				GinkgoWriter.Write([]byte(fmt.Sprintf("labels: %+v\n", node.Labels)))
				val, ok := node.Labels[labelDomainName]
				return ok && val == labelValue
			}, timeout, interval).Should(BeTrue(), "label should have been set")

		})
	})

	Context("When updating a Labels CR", func() {

	})

	Context("When deleting a Labels CR", func() {

	})

})
