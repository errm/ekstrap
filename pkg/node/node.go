package node

import (
	"github.com/errm/ekstrap/pkg/backoff"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	"log"
	"regexp"
	"time"
)

type Node struct {
	*ec2.Instance
}

type metadataClient interface {
	GetMetadata(string) (string, error)
}

var b = backoff.Backoff{[]int{1, 1, 2}}

func New(e ec2iface.EC2API, m metadataClient) (*Node, error) {
	id, err := instanceId(m)
	if err != nil {
		return nil, err
	}
	tries := 1
	for {
		output, err := e.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: []*string{id}})
		if err != nil {
			return nil, err
		}
		node := Node{output.Reservations[0].Instances[0]}
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

func (n *Node) ClusterName() string {
	re := regexp.MustCompile(`kubernetes.io\/cluster\/([\w-]+)`)
	for _, t := range n.Tags {
		if matches := re.FindStringSubmatch(*t.Key); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func instanceId(m metadataClient) (*string, error) {
	result, err := m.GetMetadata("instance-id")
	if err != nil {
		return nil, err
	}
	return &result, nil
}
