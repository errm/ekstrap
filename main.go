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
	"github.com/coreos/go-systemd/dbus"
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
	check(err)

	cluster, err := eks.Cluster(eksSvc.New(sess), instance.ClusterName())
	check(err)

	systemdDbus, err := dbus.New()
	check(err)

	systemd := &system.Systemd{systemdDbus}

	system := system.System{
		Filesystem: &file.Atomic{},
		Hostname:   systemd,
		Init:       systemd,
	}

	check(system.Configure(instance, cluster))
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
