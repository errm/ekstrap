package system

import (
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/errm/ekstrap/pkg/node"
	"io"
	"os"
)

type Filesystem interface {
	Sync(io.Reader, string, os.FileMode) error
}

type Init interface {
	RestartService(string) error
}

type Hostname interface {
	SetHostname(string) error
}

type System struct {
	Filesystem Filesystem
	Init       Init
	Hostname   Hostname
}

func (s System) Configure(n *node.Node, cluster *eks.Cluster) error {
	if err := s.Hostname.SetHostname(*n.PrivateDnsName); err != nil {
		return err
	}

	info := struct {
		Cluster *eks.Cluster
		Node    *node.Node
	}{
		Cluster: cluster,
		Node:    n,
	}

	configs, err := s.configs()
	if err != nil {
		return err
	}

	for _, config := range configs {
		config.Write(info)
	}

	if err := s.Init.RestartService("kubelet"); err != nil {
		return err
	}

	return nil
}
