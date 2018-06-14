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

	i := instance("10.6.28.199", "ip-10-6-28-199.us-west-2.compute.internal")
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

	if len(fs.files) != 3 {
		t.Errorf("expected 3 files, got %v", len(fs.files))
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
      command: /usr/local/bin/heptio-authenticator-aws
      args:
        - token
        - "-i"
        - "aws-om-cluster"`
	fs.Check(t, "/var/lib/kubelet/kubeconfig", expected, 0640)

	expected = `[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/
After=docker.service
Requires=docker.service

[Service]
ExecStart=/usr/bin/kubelet   --address=0.0.0.0   --allow-privileged=true   --cloud-provider=aws   --cluster-dns=10.100.0.10   --cluster-domain=cluster.local   --cni-bin-dir=/opt/cni/bin   --cni-conf-dir=/etc/cni/net.d   --container-runtime=docker   --node-ip=10.6.28.199   --network-plugin=cni   --cgroup-driver=cgroupfs   --register-node=true   --kubeconfig=/var/lib/kubelet/kubeconfig   --feature-gates=RotateKubeletServerCertificate=true   --anonymous-auth=false   --client-ca-file=/etc/kubernetes/pki/ca.crt

Restart=on-failure
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target`
	fs.Check(t, "/lib/systemd/system/kubelet.service", expected, 0640)

	fs.Check(t, "/etc/kubernetes/pki/ca.crt", "thisisthecertdata", 0640)

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

func instance(ip, dnsName string) *node.Node {
	return &node.Node{
		Instance: &ec2.Instance{
			PrivateIpAddress: &ip,
			PrivateDnsName:   &dnsName,
		},
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
				t.Errorf("File contents not as expected:\nactual:\n%v\n\nexpected:\n%v", actual, contents)
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
