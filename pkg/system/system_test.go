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
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/errm/ekstrap/pkg/node"
)

func TestConfigure(t *testing.T) {
	fs := &FakeFileSystem{}
	hn := &FakeHostname{}
	init := &FakeInit{}

	i := instance("10.6.28.199", "ip-10-6-28-199.us-west-2.compute.internal", 18)
	c := cluster(
		"aws-om-cluster",
		"https://74770F6B05F7A8FB0F02CFB5F7AF530C.yl4.us-west-2.eks.amazonaws.com",
		"dGhpc2lzdGhlY2VydGRhdGE=",
	)
	system := System{Filesystem: fs, Hostname: hn, Init: init}
	err := system.Configure(i, c)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if len(fs.files) != 5 {
		t.Errorf("expected 5 files, got %v", len(fs.files))
	}

	expected := `apiVersion: v1
kind: Config
clusters:
- name: aws-om-cluster
  cluster:
    server: https://74770F6B05F7A8FB0F02CFB5F7AF530C.yl4.us-west-2.eks.amazonaws.com
    certificate-authority-data: dGhpc2lzdGhlY2VydGRhdGE=
contexts:
- name: kubelet
  context:
    cluster: aws-om-cluster
    user: kubelet
current-context: kubelet
users:
- name: kubelet
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      command: aws-iam-authenticator
      args:
        - token
        - "-i"
        - "aws-om-cluster"
`
	fs.Check(t, "/var/lib/kubelet/kubeconfig", expected, 0640)

	expected = `[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/
After=docker.service
Requires=docker.service

[Service]
ExecStart=/usr/bin/kubelet \
  --address=0.0.0.0 \
  --authentication-token-webhook \
  --authorization-mode=Webhook \
  --allow-privileged=true \
  --cloud-provider=aws \
  --cluster-domain=cluster.local \
  --cni-bin-dir=/opt/cni/bin \
  --cni-conf-dir=/etc/cni/net.d \
  --container-runtime=docker \
  --network-plugin=cni \
  --cgroup-driver=cgroupfs \
  --register-node=true \
  --kubeconfig=/var/lib/kubelet/kubeconfig \
  --feature-gates=RotateKubeletServerCertificate=true \
  --anonymous-auth=false \
  --client-ca-file=/etc/kubernetes/pki/ca.crt $KUBELET_ARGS $KUBELET_MAX_PODS $KUBELET_EXTRA_ARGS

Restart=always
StartLimitInterval=0
RestartSec=5

[Install]
WantedBy=multi-user.target
`
	fs.Check(t, "/etc/systemd/system/kubelet.service", expected, 0640)

	expected = `[Service]
Environment='KUBELET_ARGS=--node-ip=10.6.28.199 --cluster-dns=172.20.0.10 --pod-infra-container-image=602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause-amd64:3.1'
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/10-kubelet-args.conf", expected, 0640)

	expected = `[Service]
Environment='KUBELET_MAX_PODS=--max-pods=18'
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/20-max-pods.conf", expected, 0640)

	expected = `thisisthecertdata
`
	fs.Check(t, "/etc/kubernetes/pki/ca.crt", expected, 0640)

	if hn.hostname != "ip-10-6-28-199.us-west-2.compute.internal" {
		t.Errorf("expected hostname to be ip-10-6-28-199.us-west-2.compute.internal, got %v", hn.hostname)
	}

	if len(init.restarted) != 1 {
		t.Errorf("expected 1 restart got %v", len(init.restarted))
	}

	if init.restarted[0] != "kubelet.service" {
		t.Errorf("expected the kubelet service to be restarted, but got %s", init.restarted[0])
	}
}

func instance(ip, dnsName string, maxPods int) *node.Node {
	return &node.Node{
		Instance: &ec2.Instance{
			PrivateIpAddress: &ip,
			PrivateDnsName:   &dnsName,
		},
		MaxPods:    maxPods,
		ClusterDNS: "172.20.0.10",
		Region:     "us-east-1",
	}
}

func cluster(name, endpoint, cert string) *eks.Cluster {
	status := eks.ClusterStatusActive
	return &eks.Cluster{
		Name:     &name,
		Endpoint: &endpoint,
		Status:   &status,
		CertificateAuthority: &eks.Certificate{
			Data: &cert,
		},
	}
}

type FakeFileSystem struct {
	files []FakeFile
}

func (f *FakeFileSystem) Sync(data io.Reader, path string, mode os.FileMode) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(data)
	log.Printf("saving a file to %v", path)
	f.files = append(f.files, FakeFile{Path: path, Contents: buf.Bytes(), Mode: mode})
	return nil
}

func (f *FakeFileSystem) Check(t *testing.T, path string, contents string, mode os.FileMode) {
	for _, file := range f.files {
		if file.Path == path {
			if file.Mode != mode {
				t.Errorf("unexpected permissions, expected %v, got %v", mode, file.Mode)
			}
			actual := string(file.Contents)
			if contents != actual {
				t.Errorf("File contents not as expected:\nactual:\n%#v\n\nexpected:\n%#v", actual, contents)
			}
			return
		}
	}
	t.Errorf("file not found: %s", path)
}

type FakeFile struct {
	Path     string
	Contents []byte
	Mode     os.FileMode
}

type FakeHostname struct {
	hostname string
}

func (h *FakeHostname) SetHostname(name string) error {
	h.hostname = name
	return nil
}

type FakeInit struct {
	restarted []string
}

func (i *FakeInit) EnsureRunning(name string) error {
	i.restarted = append(i.restarted, name)
	return nil
}
