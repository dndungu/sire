package main

import (
	"log"
	"time"

	"sire.run/pkg/k8s"

	"github.com/google/uuid"
)

func main() {
	var cluster = new(k8s.Cluster)
	cluster.Name = "example@" + time.Now().UTC().String()
	cluster.Description = "six qualities"
	cluster.Region = "us-west-2"
	cluster.CidrBlock = "172.24.0.0/16"
	cluster.Subnets = []k8s.Subnet{
		{CidrBlock: "172.24.0.0/24", Zone: "us-west-2a"},
		{CidrBlock: "172.24.1.0/24", Zone: "us-west-2b"},
		{CidrBlock: "172.24.2.0/24", Zone: "us-west-2c"},
	}
	var id = uuid.New().String()
	log.Print(id)
	cluster.Tags = []k8s.Tag{
		{Key: "Name", Value: "example"},
		{Key: "sire.run/cluster/id", Value: id},
		{Key: "kubernetes.io/cluster/example", Value: "shared"},
	}
	cluster.Policies = []k8s.Policy{
		{Description: "master node", Name: "master", Document: masterPolicy},
		{Description: "worker node", Name: "worker", Document: workerPolicy},
	}
	if err := cluster.Init(); err != nil {
		log.Fatal(err)
	}
}
