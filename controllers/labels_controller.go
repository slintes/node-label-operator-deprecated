/*
Copyright 2021.

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

package controllers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	slintesnetv1beta1 "github.com/slintes/node-label-operator/api/v1beta1"
)

// LabelsReconciler reconciles a Labels object
type LabelsReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=slintes.net.label.slintes.net,resources=labels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=slintes.net.label.slintes.net,resources=labels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=slintes.net.label.slintes.net,resources=labels/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Labels object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *LabelsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("labels", req.NamespacedName)

	// get Labels instance
	labels := &slintesnetv1beta1.Labels{}
	labelsDeleted := false
	err := r.Get(ctx, req.NamespacedName, labels)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			log.Info("Labels resource not found, will delete owned labels.")
			labelsDeleted = true
		} else {
			// Error reading the object - requeue the request.
			log.Error(err, "Failed to get Labels")
			return ctrl.Result{}, err
		}
	}

	// iterate all nodes
	// we have to
	// - remove all owned labels, if they aren't in any label rule
	// - add labels of this instance

	// we need all Labels
	allLabels := &slintesnetv1beta1.LabelsList{}
	if r.Client.List(context.TODO(), allLabels, &client.ListOptions{}); err != nil {
		log.Error(err, "Failed to list Labels")
		return ctrl.Result{}, err
	}

	// and OwnedLabels
	ownedLabels := &slintesnetv1beta1.OwnedLabelsList{}
	if r.Client.List(context.TODO(), ownedLabels, &client.ListOptions{}); err != nil {
		log.Error(err, "Failed to list OwnedLabels")
		return ctrl.Result{}, err
	}

	// get nodes
	nodes := &v1.NodeList{}
	if r.Client.List(context.TODO(), nodes, &client.ListOptions{}); err != nil {
		log.Error(err, "Failed to list Nodes")
		return ctrl.Result{}, err
	}

	// and start
	for i, nodeOrig := range nodes.Items {

		log.Info("checking node", "nodeName", nodeOrig.Name)

		node := nodeOrig.DeepCopy()
		nodeModified := false

		// check if we have owned labels on the node
		log.Info("checking owned labels")
		for labelDomainName, labelValue := range node.Labels {
			// split domainName
			parts := strings.Split(labelDomainName, "/")
			if len(parts) != 2 {
				// this should not happen...
				log.Info("Skipping unexpected label name on node", "labelName", labelDomainName, "node", node.Name)
				continue
			}
			labelDomain := parts[0]
			labelName := parts[1]

			// check if we own this label
			for _, ownedLabel := range ownedLabels.Items {
				log.Info("checking owned label", "label", ownedLabel.Spec)
				if ownedLabel.Spec.Domain != nil && *ownedLabel.Spec.Domain != labelDomain {
					// domain set but doesn't match, move on
					log.Info("  domain does not match", "nodeDomain", labelDomain, "ownedDomain", ownedLabel.Spec.Domain)
					continue
				}
				if ownedLabel.Spec.NamePattern != nil {
					match, err := regexp.MatchString(*ownedLabel.Spec.NamePattern, labelName)
					if err != nil {
						log.Error(err, "invalid regular expression, moving on to next owned label", "pattern", ownedLabel.Spec.NamePattern)
						continue
					}
					if !match {
						// name pattern set but doesn't match, move on
						log.Info("  name pattern does not match")
						continue
					}
				}

				log.Info("  we own it! checking rules")

				// we own this label
				// check if it is still covered by a label rule
				labelCovered := false
			CoveredLoop:
				for _, rules := range allLabels.Items {
					for _, rule := range rules.Spec.Rules {
						for _, ruleLabel := range rule.Labels {
							// split to domain/name and value
							parts := strings.Split(ruleLabel, "=")
							if len(parts) != 2 {
								log.Info("skipping unexpected rule label", ruleLabel)
								continue
							}
							if parts[0] == labelDomainName && parts[1] == labelValue {

								log.Info("    label matches...")

								// label matches... does the node?
								for _, nodeNamePattern := range rule.NodeNamePatterns {
									match, err := regexp.MatchString(nodeNamePattern, node.Name)
									if err != nil {
										log.Error(err, "invalid regular expression, moving on to next rule")
										continue
									}
									if match {
										// label is still valid!
										// break out of this nested loops
										log.Info("    and value matches! keeping label")
										labelCovered = true
										break CoveredLoop
									}
								}
							}
						}
					}
				}
				if !labelCovered {
					// we need to remove the label
					log.Info("  deleting uncovered owned label!")
					nodeLabels := node.Labels
					delete(nodeLabels, labelDomainName)
					node.Labels = nodeLabels
					nodeModified = true
				}
			}
		}

		// owned labels are removed now on this node
		// add new labels
		if !labelsDeleted {
			for _, rule := range labels.Spec.Rules {
				for _, nodeNamePattern := range rule.NodeNamePatterns {
					match, err := regexp.MatchString(nodeNamePattern, node.Name)
					if err != nil {
						log.Error(err, "invalid regular expression, moving on to next rule")
						continue
					}
					if !match {
						continue
					}
					// we have a match, add labels!
					for _, label := range rule.Labels {
						// split to domain/name and value
						parts := strings.Split(label, "=")
						if len(parts) != 2 {
							log.Info("invalid label, less or more than one \"=\", moving on to next rule", "label", label)
							continue
						}
						log.Info("adding label to node based on pattern", "label", label, "nodeName", node.Name, "pattern", nodeNamePattern)
						nodes.Items[i].Labels[parts[0]] = parts[1]
						nodeModified = true
					}
				}
			}
		}

		// save node
		if nodeModified {
			baseToPatch := client.MergeFrom(&nodes.Items[i])
			if err := r.Client.Patch(context.TODO(), node, baseToPatch); err != nil {
				log.Error(err, "Failed to patch Node")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LabelsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&slintesnetv1beta1.Labels{}).
		Complete(r)
}
