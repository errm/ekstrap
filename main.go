package main

import (
	"log"

	"github.com/errm/ekstrap/pkg/eks"
	"github.com/errm/ekstrap/pkg/file"
	"github.com/errm/ekstrap/pkg/node"
	"github.com/errm/ekstrap/pkg/system"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	eksSvc "github.com/aws/aws-sdk-go/service/eks"
)

var metadata = ec2metadata.New(session.Must(session.NewSession()))
var sess = session.Must(session.NewSession(&aws.Config{Region: region()}))

func region() *string {
	region, err := metadata.Region()
	if err != nil {
		log.Fatal(err)
	}
	return &region
}

func main() {
	instance, err := node.New(ec2.New(sess), metadata)
	if err != nil {
		log.Fatal(err)
	}

	cluster, err := eks.Cluster(eksSvc.New(sess), instance.ClusterName())
	if err != nil {
		log.Fatal(err)
	}

	system := system.System{
		Filesystem: &file.Atomic{},
		Hostname:   &system.Systemd{},
		Init:       &system.Systemd{},
	}

	if err := system.Configure(instance, cluster); err != nil {
		log.Fatal(err)
	}
}
