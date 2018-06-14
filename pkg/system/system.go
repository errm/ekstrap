package system

import (
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/errm/ekstrap/pkg/node"

	"bytes"
	"encoding/base64"
	"io"
	"os"
	"text/template"
)

type Filesystem interface {
	Sync(io.Reader, string, os.FileMode) error
}

type Init interface {
	EnsureRunning(string) error
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
		config.write(info)
	}

	return s.Init.EnsureRunning("kubelet")
}

func (s System) configs() ([]config, error) {
	configs := []config{}
	for path, content := range defaultTemplates {
		template, err := template.New(path).Funcs(template.FuncMap{"b64dec": base64decode}).Parse(content)
		if err != nil {
			return configs, err
		}
		configs = append(configs, config{
			template:   template,
			path:       path,
			filesystem: s.Filesystem,
		})
	}
	return configs, nil
}

func base64decode(v string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type config struct {
	template   *template.Template
	path       string
	filesystem Filesystem
}

func (c config) write(data interface{}) error {
	var buff bytes.Buffer
	err := c.template.Execute(&buff, data)
	if err != nil {
		return err
	}
	return c.filesystem.Sync(&buff, c.path, 0640)
}
