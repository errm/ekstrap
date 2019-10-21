/*
Copyright 2018 Edward Robinson.

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

package node

import (
	"github.com/errm/ekstrap/pkg/backoff"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	"fmt"
	"log"
	"regexp"
	"sort"
	"time"
)

// Node represents and EC2 instance.
type Node struct {
	*ec2.Instance
	Region           string
	ContainerRuntime string
}

type metadataClient interface {
	GetMetadata(string) (string, error)
}

var b = backoff.Backoff{Seq: []int{1, 1, 2}}

// New returns a Node instance.
//
// If the EC2 instance doesn't have the expected kubernetes tag, it will backoff and retry.
// If it isn't able to query EC2 or there are any other errors, an error will be returned.
func New(e ec2iface.EC2API, m metadataClient, region *string, containerRuntime string) (*Node, error) {
	id, err := instanceID(m)
	if err != nil {
		return nil, err
	}
	tries := 1
	for {
		output, err := e.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: []*string{id}})
		if err != nil {
			return nil, err
		}
		instance := output.Reservations[0].Instances[0]
		node := Node{
			Instance:         instance,
			Region:           *region,
			ContainerRuntime: containerRuntime,
		}
		if node.ClusterName() == "" {
			sleepFor := b.Duration(tries)
			log.Printf("The kubernetes.io/cluster/<name> tag is not yet set, will try again in %s", sleepFor)
			time.Sleep(sleepFor)
			tries++
			continue
		}
		return &node, nil
	}
}

// ClusterName returns the cluster name.
//
// It reads the cluster name from a tag on the EC2 instance.
func (n *Node) ClusterName() string {
	re := regexp.MustCompile(`kubernetes.io\/cluster\/([\w-]+)`)
	for _, t := range n.Tags {
		if matches := re.FindStringSubmatch(*t.Key); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

// Labels returns list of kubernetes labels for this node
//
// If the node is a spot instance the node-role.kubernetes.io/spot-worker label
// will be set, otherwise the node-role.kubernetes.io/worker is set.
//
// Other custom labels can be set using EC2 tags with the k8s.io/cluster-autoscaler/node-template/label/ prefix
func (n *Node) Labels() []string {
	labels := make(map[string]string)

	if n.Spot() {
		labels["node-role.kubernetes.io/spot-worker"] = "true"
	} else {
		labels["node-role.kubernetes.io/worker"] = "true"
	}

	re := regexp.MustCompile(`k8s.io\/cluster-autoscaler\/node-template\/label\/(.*)`)
	for _, t := range n.Tags {
		if matches := re.FindStringSubmatch(*t.Key); len(matches) == 2 {
			labels[matches[1]] = *t.Value
		}
	}

	l := make([]string, 0, len(labels))
	for key, value := range labels {
		l = append(l, key+"="+value)
	}
	sort.Strings(l)
	return l
}

// Spot returns true is this node is a spot instance
func (n *Node) Spot() bool {
	if n.InstanceLifecycle != nil && *n.InstanceLifecycle == ec2.InstanceLifecycleTypeSpot {
		return true
	}
	return false
}

// Taints returns a list of kuberntes taints for this node
//
// Taints can be set using EC2 tags with the k8s.io/cluster-autoscaler/node-template/taint/ prefix
func (n *Node) Taints() []string {
	var taints []string
	re := regexp.MustCompile(`k8s.io\/cluster-autoscaler\/node-template\/taint\/(.*)`)
	for _, t := range n.Tags {
		if matches := re.FindStringSubmatch(*t.Key); len(matches) == 2 {
			taints = append(taints, matches[1]+"="+*t.Value)
		}
	}
	sort.Strings(taints)
	return taints
}

func instanceID(m metadataClient) (*string, error) {
	result, err := m.GetMetadata("instance-id")
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// MaxPods returns the maximum number of pods that can be scheduled to this node
//
// see https://github.com/aws/amazon-vpc-cni-k8s#setup for more info
func (n *Node) MaxPods() int {
	enis := InstanceENIsAvailable[*n.InstanceType]
	ips := InstanceIPsAvailable[*n.InstanceType]
	if ips == 0 {
		return 0
	}
	return enis * (ips - 1)
}

// ReservedCPU returns the CPU in millicores that should be reserved for Kuberntes own use on this node
//
// The calculation here is based on information found in the GKE documentation
// here: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
// I think that it should also apply to AWS
func (n *Node) ReservedCPU() string {
	cores := InstanceCores[*n.InstanceType]
	reserved := 0.0
	for core := 1; core <= cores; core++ {
		switch core {
		case 1:
			reserved += 60.0
		case 2:
			reserved += 10.0
		case 3, 4:
			reserved += 5.0
		default:
			reserved += 2.5
		}
	}
	if reserved == 0.0 {
		log.Printf("The number of CPU cores is unknown for the %s instance type, --kube-reserved will not be configured", *n.InstanceType)
		return ""
	}
	return fmt.Sprintf("%.0fm", reserved)
}

// ReservedMemory returns the memory that should be reserved for Kuberntes own use on this node
//
// The calculation here is based on information found in the GKE documentation
// here: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
// I think that it should also apply to AWS
func (n *Node) ReservedMemory() string {
	memory := InstanceMemory[*n.InstanceType]
	reserved := 0.0
	for i := 0; i < memory; i++ {
		switch {
		case i < 4096:
			reserved += 0.25
		case i < 8192:
			reserved += 0.2
		case i < 16384:
			reserved += 0.1
		case i < 131072:
			reserved += 0.06
		default:
			reserved += 0.02
		}
	}
	if reserved == 0.0 {
		log.Printf("The Memory of the %s instance type is unknown, --kube-reserved will not be configured", *n.InstanceType)
		return ""
	}
	return fmt.Sprintf("%.0fMi", reserved)
}

// ClusterDNS returns the in cluster IP address that kube-dns should avalible at
func (n *Node) ClusterDNS() string {
	if n.PrivateIpAddress != nil && len(*n.PrivateIpAddress) > 3 && (*n.PrivateIpAddress)[0:3] == "10." {
		return "172.20.0.10"
	}
	return "10.100.0.10"
}

const (
	Arm64 = "arm64"
	Amd64 = "x86_64"
)

func (n *Node) PauseImage() string {
	var account, arch string
	switch n.Region {
	case "ap-east-1":
		account = "800184023465"
	case "me-south-1":
		account = "558608220178"
	default:
		account = "602401143452"
	}

	switch *n.Architecture {
	case Amd64:
		arch = "amd64"
	case Arm64:
		arch = "arm64"
	default:
		panic(fmt.Sprintf("%s is not a supported machine architecture", *n.Architecture))
	}

	return account + ".dkr.ecr." + n.Region + ".amazonaws.com/eks/pause-" + arch + ":3.1"
}
