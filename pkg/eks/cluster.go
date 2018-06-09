package eks

import (
	"fmt"
	"log"
	"time"

	"github.com/errm/ekstrap/pkg/backoff"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

var b = backoff.Backoff{[]int{1, 3, 4, 8, 16, 32, 64}}

func Cluster(svc eksiface.EKSAPI, name string) (*eks.Cluster, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(name),
	}
	tries := 1
	for {
		result, err := svc.DescribeCluster(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceNotFoundException:
					sleepFor := b.Duration(tries)
					log.Printf("The EKS cluster: %s does not (yet) exist, will try again in %s", name, sleepFor)
					time.Sleep(sleepFor)
					tries++
					continue
				case eks.ErrCodeServiceUnavailableException:
					sleepFor := b.Duration(tries)
					log.Printf("The EKS service is currentlty unavalible, will try again in %s", sleepFor)
					time.Sleep(sleepFor)
					tries++
					continue
				}
			}
			return nil, err
		}
		cluster := result.Cluster
		switch *cluster.Status {
		case eks.ClusterStatusActive:
			return result.Cluster, nil
		case eks.ClusterStatusCreating:
			sleepFor := b.Duration(tries)
			log.Printf("Waiting for the EKS cluster: %s to start, will try again in about %d seconds...", name, sleepFor)
			time.Sleep(sleepFor)
			tries++
			continue
		}
		return nil, fmt.Errorf("Cannot use the EKS cluster: %s, becuase it is %s", name, *cluster.Status)
	}
}
