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

package main

//go:generate packr

import (
	"log"

	"github.com/errm/ekstrap/pkg/eks"
	"github.com/errm/ekstrap/pkg/file"
	"github.com/errm/ekstrap/pkg/node"
	"github.com/errm/ekstrap/pkg/system"
	"github.com/errm/ekstrap/pkg/util"

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
	if err != nil || !util.IsAWSRegion(region) {
		log.Fatal("I don't seem to be running on an AWS EC2 instance. Can't reach the ec2 metadata service, or it's output seems to be invalid!")
	}
	return &region
}

func main() {
	instance, err := node.New(ec2.New(sess), metadata, region())
	check(err)

	cluster, err := eks.Cluster(eksSvc.New(sess), instance.ClusterName())
	check(err)

	systemdDbus, err := dbus.New()
	check(err)

	systemd := &system.Systemd{Conn: systemdDbus}

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
