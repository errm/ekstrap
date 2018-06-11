package system

import (
	"bytes"
	"encoding/base64"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/errm/ekstrap/pkg/node"
	"io"
	"os"
	"strings"
	"text/template"
)

type Files interface {
	Sync(io.Reader, string, os.FileMode) error
}

type Init interface {
	RestartService(string) error
}

type Hostname interface {
	SetHostname(string) error
}

type System struct {
	Files    Files
	Init     Init
	Hostname Hostname
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

	template, err := kubeconfig()
	if err != nil {
		return err
	}

	if err := s.writeConfig("/var/lib/kubelet/kubeconfig", template, info); err != nil {
		return err
	}

	template, err = kubeletService()
	if err != nil {
		return err
	}

	if err := s.writeConfig("/lib/systemd/system/kubelet.service", template, info); err != nil {
		return err
	}

	if err := s.writeCertificate("/etc/kubernetes/pki/ca.crt", *cluster.CertificateAuthority.Data); err != nil {
		return err
	}

	if err := s.Init.RestartService("kubelet"); err != nil {
		return err
	}

	return nil
}

func (s System) writeConfig(path string, templ *template.Template, data interface{}) error {
	var buff bytes.Buffer
	err := templ.Execute(&buff, data)
	if err != nil {
		return err
	}
	return s.Files.Sync(&buff, path, 0640)
}

func (s System) writeCertificate(path string, data string) error {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))
	return s.Files.Sync(decoder, path, 0640)
}

func kubeconfig() (*template.Template, error) {
	return template.New("kubeconfig").Parse(
		`apiVersion: v1
kind: Config
clusters:
- name: {{.Cluster.Name}}
  cluster:
    server: {{.Cluster.Endpoint}}
    certificate-authority-data: {{.Cluster.CertificateAuthority.Data}}
contexts:
- name: kubelet
  context:
    cluster: {{.Cluster.Name}}
    user: kubelet
current-context: kubelet
users:
- name: kubelet
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      command: /usr/local/bin/heptio-authenticator-aws
      args:
        - token
        - "-i"
        - "{{.Cluster.Name}}"
`)
}

func kubeletService() (*template.Template, error) {
	return template.New("kubelet-service").Parse(
		`[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/
After=docker.service
Requires=docker.service

[Service]
ExecStart=/usr/bin/kubelet   --address=0.0.0.0   --allow-privileged=true   --cloud-provider=aws   --cluster-dns=10.100.0.10   --cluster-domain=cluster.local   --cni-bin-dir=/opt/cni/bin   --cni-conf-dir=/etc/cni/net.d   --container-runtime=docker   --node-ip={{.Node.PrivateIpAddress}}   --network-plugin=cni   --cgroup-driver=cgroupfs   --register-node=true   --kubeconfig=/var/lib/kubelet/kubeconfig   --feature-gates=RotateKubeletServerCertificate=true   --anonymous-auth=false   --client-ca-file=/etc/kubernetes/pki/ca.crt

Restart=on-failure
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`)
}
