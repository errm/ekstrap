package eks

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

func Cluster(svc eksiface.EKSAPI, name string) (*eks.Cluster, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(name),
	}
	backoff := 1
	for {
		result, err := svc.DescribeCluster(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceNotFoundException:
					log.Printf("The EKS cluster: %s does not (yet) exist, will try again in %d seconds...", name, backoff)
					time.Sleep(time.Duration(backoff) * time.Second)
					if backoff < 64 {
						backoff = backoff * 2
					}
					continue
				case eks.ErrCodeServiceUnavailableException:
					if backoff > 512 {
						return nil, err
					}
					log.Printf("The EKS service is currentlty unavalible, will try again in %d seconds...", backoff)
					time.Sleep(time.Duration(backoff) * time.Second)
					backoff = backoff * 2
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
			log.Printf("Waiting for the EKS cluster: %s to start, will try again in %d seconds...", name, backoff)
			time.Sleep(time.Duration(backoff) * time.Second)
			if backoff < 64 {
				backoff = backoff * 2
			}
			continue
		}
		return nil, fmt.Errorf("Cannot use the EKS cluster: %s, becuase it is %s", name, *cluster.Status)
	}
}
