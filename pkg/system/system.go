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

package system

import (
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/errm/ekstrap/pkg/node"
	"github.com/gobuffalo/packr/v2"

	"bytes"
	"encoding/base64"
	"io"
	"os"
	"text/template"
)

type filesystem interface {
	Sync(io.Reader, string, os.FileMode) error
}

type initsystem interface {
	EnsureRunning(string) error
}

type hostname interface {
	SetHostname(string) error
}

// System represents the system we are configuring and
// should be created with the interfaces to interact with it
type System struct {
	Filesystem filesystem
	Init       initsystem
	Hostname   hostname
}

// Configure configures the system to connect to the EKS cluster given the node
// and cluster metadata provided as arguments
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
		if config.write(info) != nil {
			return err
		}
	}

	return s.Init.EnsureRunning("kubelet.service")
}

func (s System) configs() ([]config, error) {
	configs := []config{}
	box := packr.New("system templates", "./templates")
	err := box.Walk(func(path string, f packr.File) error {
		template, err := template.New(path).Funcs(template.FuncMap{"b64dec": base64decode}).Parse(f.String())
		configs = append(configs, config{
			template:   template,
			path:       "/" + path,
			filesystem: s.Filesystem,
		})
		return err
	})
	return configs, err
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
	filesystem filesystem
}

func (c config) write(data interface{}) error {
	var buff bytes.Buffer
	err := c.template.Execute(&buff, data)
	if err != nil {
		return err
	}
	return c.filesystem.Sync(&buff, c.path, 0640)
}
