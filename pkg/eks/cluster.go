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

var b = backoff.Backoff{Seq: []int{1, 2, 4, 8, 16, 32, 64}}

// Cluster returns the named EKS cluster.
//
// If the cluster doesn't exist, or hasn't yet started it will block until it is ready.
// If the EKS service is unavalible it will backoff and retry
// If the cluster is deleting failed, or there are any other errors an error will be returned
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
			log.Printf("Waiting for the EKS cluster: %s to start, will try again in %s", name, sleepFor)
			time.Sleep(sleepFor)
			tries++
			continue
		}
		return nil, fmt.Errorf("Cannot use the EKS cluster: %s, becuase it is %s", name, *cluster.Status)
	}
}
