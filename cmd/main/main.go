package main

import (
	"log"

	"zc/pkg/k8s"

	"github.com/google/uuid"
)

func main() {
	var cluster = new(k8s.Cluster)
	cluster.Region = "us-west-2"
	cluster.CidrBlock = "172.24.0.0/16"
	cluster.Subnets = []k8s.Subnet{
		{CidrBlock: "172.24.0.0/24", Zone: "us-west-2a"},
		{CidrBlock: "172.24.1.0/24", Zone: "us-west-2b"},
		{CidrBlock: "172.24.2.0/24", Zone: "us-west-2c"},
	}
	var id = uuid.New().String()
	cluster.Tags = []k8s.Tag{
		{Key: "Name", Value: "dndungu-etcd"},
		{Key: "kubernetes.io/cluster/dndungu-etcd", Value: "owned"},
		{Key: "dndungu-id", Value: id},
	}
	cluster.Policies = []k8s.Policy{
		{Description: "dndungu-etcd-control-plane", Name: "dndungu-control-plane", Document: controlPlanePolicy},
		{Description: "dndungu-etcd-node", Name: "dndungu-node", Document: nodePolicy},
	}
	log.Print(id)
	if err := cluster.Init(); err != nil {
		log.Fatal(err)
	}
}
