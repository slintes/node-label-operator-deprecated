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
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var nodelog = logf.Log.WithName("nodes-webhook")

// +kubebuilder:webhook:path=/label-v1-nodes,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",resources=nodes,verbs=create;update,versions=v1,name=mnode.kb.io,admissionReviewVersions={v1,v1beta1}

// NodeLabeler adds labels to Nodes
type NodeLabeler struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (n *NodeLabeler) Handle(ctx context.Context, req admission.Request) admission.Response {

	nodelog.Info("node webhook is called!")

	node := &v1.Node{}
	err := n.decoder.Decode(req, node)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	nodelog.Info("node is decoded", "node", fmt.Sprintf("%+v", node))

	if node.Labels == nil {
		node.Labels = map[string]string{}
	}
	node.Labels["my-node-webhook"] = "works"

	marshaledNode, err := json.Marshal(node)
	if err != nil {
		nodelog.Error(err, "marshalling response went wrong")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledNode)
}

// InjectDecoder injects the decoder.
func (n *NodeLabeler) InjectDecoder(d *admission.Decoder) error {
	n.decoder = d
	return nil
}

func (n *NodeLabeler) SetupWebhookWithManager(mgr ctrl.Manager) {
	hookServer := mgr.GetWebhookServer()
	hookServer.Register("/label-v1-nodes", &webhook.Admission{Handler: &NodeLabeler{Client: mgr.GetClient()}})
}
