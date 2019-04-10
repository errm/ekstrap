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

	i := instance(map[string]string{}, false, "docker")
	c := cluster()
	system := System{Filesystem: fs, Hostname: hn, Init: init}
	err := system.Configure(i, c)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if len(fs.files) != 8 {
		t.Errorf("expected 8 files, got %v", len(fs.files))
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
ExecStartPre=/sbin/iptables -P FORWARD ACCEPT
ExecStart=/usr/bin/kubelet \
  --allow-privileged=true \
  --cloud-provider=aws \
  --config=/etc/kubernetes/kubelet/config.yaml \
  --network-plugin=cni \
  --kubeconfig=/var/lib/kubelet/kubeconfig $KUBELET_CONTAINER_RUNTIME_ARGS $KUBELET_ARGS $KUBELET_NODE_LABELS $KUBELET_NODE_TAINTS $KUBELET_EXTRA_ARGS

Restart=on-failure
RestartForceExitStatus=SIGPIPE
RestartSec=5
KillMode=process

[Install]
WantedBy=multi-user.target
`
	fs.Check(t, "/etc/systemd/system/kubelet.service", expected, 0640)

	expected = `[Service]
Environment='KUBELET_ARGS=--node-ip=10.6.28.199 --pod-infra-container-image=602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause-amd64:3.1'
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/10-kubelet-args.conf", expected, 0640)

	expected = `kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
address: 0.0.0.0
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: "/etc/kubernetes/pki/ca.crt"
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
clusterDomain: cluster.local
hairpinMode: hairpin-veth
clusterDNS: [172.20.0.10]
cgroupDriver: cgroupfs
cgroupRoot: /
featureGates:
  RotateKubeletServerCertificate: true
serverTLSBootstrap: true
serializeImagePulls: false
kubeReserved:
  cpu: 70m
  memory: 1024Mi
maxPods: 27
evictionHard:
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
`
	fs.Check(t, "/etc/kubernetes/kubelet/config.yaml", expected, 0640)

	expected = `[Service]
Environment='KUBELET_NODE_LABELS=--node-labels="node-role.kubernetes.io/worker=true"'
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/20-labels.conf", expected, 0640)

	expected = `[Service]`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/30-taints.conf", expected, 0640)

	expected = `[Service]
Environment="KUBELET_CONTAINER_RUNTIME_ARGS=--container-runtime=docker"
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/40-container-runtime.conf", expected, 0640)

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

func TestConfigureSpotInstanceLabels(t *testing.T) {
	fs := &FakeFileSystem{}
	hn := &FakeHostname{}
	init := &FakeInit{}

	tags := map[string]string{
		"node-role.kubernetes.io/worker": "true",
	}

	i := instance(tags, true, "docker")
	c := cluster()
	system := System{Filesystem: fs, Hostname: hn, Init: init}
	err := system.Configure(i, c)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	expected := `[Service]
Environment='KUBELET_NODE_LABELS=--node-labels="node-role.kubernetes.io/spot-worker=true"'
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/20-labels.conf", expected, 0640)
}

func TestConfigureLabels(t *testing.T) {
	fs := &FakeFileSystem{}
	hn := &FakeHostname{}
	init := &FakeInit{}

	tags := map[string]string{
		"k8s.io/cluster-autoscaler/node-template/label/gpu-type": "K80",
	}

	i := instance(tags, false, "docker")
	c := cluster()
	system := System{Filesystem: fs, Hostname: hn, Init: init}
	err := system.Configure(i, c)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	expected := `[Service]
Environment='KUBELET_NODE_LABELS=--node-labels="gpu-type=K80,node-role.kubernetes.io/worker=true"'
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/20-labels.conf", expected, 0640)
}

func TestConfigureTaints(t *testing.T) {
	fs := &FakeFileSystem{}
	hn := &FakeHostname{}
	init := &FakeInit{}

	tags := map[string]string{
		"k8s.io/cluster-autoscaler/node-template/taint/node-role.kubernetes.io/worker": "true:PreferNoSchedule",
	}

	i := instance(tags, false, "docker")
	c := cluster()
	system := System{Filesystem: fs, Hostname: hn, Init: init}
	err := system.Configure(i, c)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	expected := `[Service]
Environment='KUBELET_NODE_TAINTS=--register-with-taints="node-role.kubernetes.io/worker=true:PreferNoSchedule"'
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/30-taints.conf", expected, 0640)
}

func TestContainerd(t *testing.T) {
	fs := &FakeFileSystem{}
	hn := &FakeHostname{}
	init := &FakeInit{}

	i := instance(map[string]string{}, false, "containerd")
	c := cluster()
	system := System{Filesystem: fs, Hostname: hn, Init: init}
	err := system.Configure(i, c)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	expected := `[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/
After=containerd.service
Requires=containerd.service

[Service]
ExecStartPre=/sbin/iptables -P FORWARD ACCEPT
ExecStart=/usr/bin/kubelet \
  --allow-privileged=true \
  --cloud-provider=aws \
  --config=/etc/kubernetes/kubelet/config.yaml \
  --network-plugin=cni \
  --kubeconfig=/var/lib/kubelet/kubeconfig $KUBELET_CONTAINER_RUNTIME_ARGS $KUBELET_ARGS $KUBELET_NODE_LABELS $KUBELET_NODE_TAINTS $KUBELET_EXTRA_ARGS

Restart=on-failure
RestartForceExitStatus=SIGPIPE
RestartSec=5
KillMode=process

[Install]
WantedBy=multi-user.target
`
	fs.Check(t, "/etc/systemd/system/kubelet.service", expected, 0640)

	expected = `[Service]
Environment="KUBELET_CONTAINER_RUNTIME_ARGS=--container-runtime=remote --runtime-request-timeout=15m --container-runtime-endpoint=unix:///run/containerd/containerd.sock"
`
	fs.Check(t, "/etc/systemd/system/kubelet.service.d/40-container-runtime.conf", expected, 0640)
}

func instance(tags map[string]string, spot bool, runtime string) *node.Node {
	ip := "10.6.28.199"
	dnsName := "ip-10-6-28-199.us-west-2.compute.internal"
	var ec2tags []*ec2.Tag
	for key, value := range tags {
		key := key
		value := value
		ec2tags = append(ec2tags, &ec2.Tag{
			Key:   &key,
			Value: &value,
		})
	}
	var instanceLifecycle *string
	if spot {
		il := ec2.InstanceLifecycleTypeSpot
		instanceLifecycle = &il
	}
	instanceType := "c5.large"
	return &node.Node{
		Instance: &ec2.Instance{
			PrivateIpAddress:  &ip,
			PrivateDnsName:    &dnsName,
			Tags:              ec2tags,
			InstanceType:      &instanceType,
			InstanceLifecycle: instanceLifecycle,
		},
		Region:           "us-east-1",
		ContainerRuntime: runtime,
	}
}

func cluster() *eks.Cluster {
	name := "aws-om-cluster"
	endpoint := "https://74770F6B05F7A8FB0F02CFB5F7AF530C.yl4.us-west-2.eks.amazonaws.com"
	cert := "dGhpc2lzdGhlY2VydGRhdGE="
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
	if _, err := buf.ReadFrom(data); err != nil {
		return err
	}
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
