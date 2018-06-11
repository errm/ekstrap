package node

import (
	"github.com/errm/ekstrap/pkg/backoff"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	"log"
	"regexp"
	"time"
)

// Node represents and EC2 instance.
type Node struct {
	*ec2.Instance
}

type metadataClient interface {
	GetMetadata(string) (string, error)
}

var b = backoff.Backoff{Seq: []int{1, 1, 2}}

// New returns a Node instance.
//
// If the EC2 instance doesn't have the expected kubernetes tag, it will backoff and retry.
// If it isn't able to query EC2 or there are any other errors, an error will be returned.
func New(e ec2iface.EC2API, m metadataClient) (*Node, error) {
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
