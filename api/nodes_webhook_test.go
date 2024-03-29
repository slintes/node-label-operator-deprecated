package api

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/slintes/node-label-operator/api/v1beta1"
	. "github.com/slintes/node-label-operator/pkg/test"
)

var _ = Describe("Nodes webhook", func() {

	When("Creating a node", func() {

		var nodeNotMatching *v1.Node
		var nodeMatching *v1.Node
		var labels *v1beta1.Labels

		BeforeEach(func() {
			// Order matters!
			// And it is important Labels exists for sure before nodes are created
			By("Creating a Labels CR")
			labels = GetLabels()
			Expect(k8sClient.Create(ctx, labels)).Should(Succeed(), "labels should have been created")
			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKeyFromObject(labels), labels)
			}, Timeout, Interval).Should(Succeed(), "labels should exist")

			By("Creating nodes")
			nodeNotMatching = GetNode(NodeNameNoMatch)
			Expect(k8sClient.Create(context.Background(), nodeNotMatching)).Should(Succeed(), "nodeNotMatching should have been created")
			nodeMatching = GetNode(NodeNameMatching)
			Expect(k8sClient.Create(context.Background(), nodeMatching)).Should(Succeed(), "nodeMatching should have been created")
		})

		AfterEach(func() {
			By("Cleaning up nodes and labels")
			Expect(k8sClient.Delete(context.Background(), nodeNotMatching)).Should(Succeed(), "nodeNotMatching should have been deleted")
			Expect(k8sClient.Delete(context.Background(), nodeMatching)).Should(Succeed(), "nodeMatching should have been deleted")

			Expect(k8sClient.Delete(context.Background(), labels)).Should(Succeed(), "labels should have been deleted")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(labels), labels)
				return err != nil && errors.IsNotFound(err)
			}, Timeout, Interval).Should(BeTrue(), "labels should be away")
		})

		It("Should add labels when node matches", func() {
			By("Verifying that label was set on matching node")
			Eventually(func() bool {
				Expect(k8sClient.Get(context.Background(), client.ObjectKeyFromObject(nodeMatching), nodeMatching)).Should(Succeed())
				GinkgoWriter.Write([]byte(fmt.Sprintf("labels: %+v\n", nodeMatching.Labels)))
				val, ok := nodeMatching.Labels[LabelDomainName]
				return ok && val == LabelValue
			}, Timeout, Interval).Should(BeTrue(), "label should have been set")
		})

		It("Should not add labels when node not matches", func() {
			By("Verifying that label was not set on not matching node")
			Consistently(func() bool {
				Expect(k8sClient.Get(context.Background(), client.ObjectKeyFromObject(nodeNotMatching), nodeNotMatching)).Should(Succeed())
				GinkgoWriter.Write([]byte(fmt.Sprintf("labels: %+v\n", nodeNotMatching.Labels)))
				_, ok := nodeNotMatching.Labels[LabelDomainName]
				return ok
			}, Timeout, Interval).Should(BeFalse(), "label should not have been set")
		})

	})

})
