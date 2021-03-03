package controllers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1

	nodeNamePattern  = "node-match-.*"
	nodeNameMatching = "node-match-yes"
	nodeNameNoMatch  = "node-no-match"

	labelDomain     = "test.slintes.net"
	labelName       = "foo1"
	labelValue      = "bar1"
	labelDomainName = labelDomain + "/" + labelName
	label           = labelDomainName + "=" + labelValue
)

var ctx = context.Background()

func getNode(name string) *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
