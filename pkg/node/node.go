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
	"time"
)

// Node represents and EC2 instance.
type Node struct {
	*ec2.Instance
	MaxPods        int
	ReservedCPU    string
	ReservedMemory string
	ClusterDNS     string
	Region         string
}

type metadataClient interface {
	GetMetadata(string) (string, error)
}

var b = backoff.Backoff{Seq: []int{1, 1, 2}}

// New returns a Node instance.
//
// If the EC2 instance doesn't have the expected kubernetes tag, it will backoff and retry.
// If it isn't able to query EC2 or there are any other errors, an error will be returned.
func New(e ec2iface.EC2API, m metadataClient, region *string) (*Node, error) {
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
		node := Node{Instance: instance, MaxPods: maxPods(instance.InstanceType), ReservedCPU: reservedCPU(instance.InstanceType), ReservedMemory: reservedMemory(instance.InstanceType), ClusterDNS: clusterDNS(instance.PrivateIpAddress), Region: *region}
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

func instanceID(m metadataClient) (*string, error) {
	result, err := m.GetMetadata("instance-id")
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func maxPods(instanceType *string) int {
	enis := InstanceENIsAvailable[*instanceType]
	ips := InstanceIPsAvailable[*instanceType]
	if ips == 0 {
		return 0
	}
	return enis * (ips - 1)
}

// The calculation here is based on information found in the GKE documentation
// here: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
// I think that it should also apply to AWS
func reservedCPU(instanceType *string) string {
	cores := InstanceCores[*instanceType]
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
		log.Printf("The number of CPU cores is unknown for the %s instance type, --kube-reserved will not be configured", *instanceType)
		return ""
	}
	return fmt.Sprintf("%.0fm", reserved)
}

// The calculation here is based on information found in the GKE documentation
// here: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
// I think that it should also apply to AWS
func reservedMemory(instanceType *string) string {
	memory := InstanceMemory[*instanceType]
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
		log.Printf("The Memory of the %s instance type is unknown, --kube-reserved will not be configured", *instanceType)
		return ""
	}
	return fmt.Sprintf("%.0fMi", reserved)
}

func clusterDNS(ip *string) string {
	if ip != nil && len(*ip) > 3 && (*ip)[0:3] == "10." {
		return "172.20.0.10"
	}
	return "10.100.0.10"
}
