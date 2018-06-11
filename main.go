package main

import (
	"bytes"
	"encoding/base64"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"

	"github.com/errm/ekstrap/pkg/eks"
	"github.com/errm/ekstrap/pkg/file"
	"github.com/errm/ekstrap/pkg/node"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	eksSvc "github.com/aws/aws-sdk-go/service/eks"
)

var kubeconfig = template.Must(template.New("kubeconfig").Parse(
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
`))

var kubeletService = template.Must(template.New("kubelet-service").Parse(
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
`))

func writeConfig(path string, templ *template.Template, data interface{}) error {
	var buff bytes.Buffer
	err := templ.Execute(&buff, data)
	if err != nil {
		return err
	}
	return file.Sync(&buff, path, 0640)
}

func writeCertificate(path string, data string) error {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))
	return file.Sync(decoder, path, 0640)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func setHostname(hostname string) error {
	if currHostname, err := os.Hostname(); err != nil || currHostname == hostname {
		return err
	}
	h := []byte(hostname)
	log.Printf("setting hostname to %s", h)
	if err := file.Sync(bytes.NewReader(h), "/etc/hostname", 0644); err != nil {
		return err
	}
	return syscall.Sethostname(h)
}

func main() {
	metadata := ec2metadata.New(session.Must(session.NewSession()))
	region, err := metadata.Region()
	if err != nil {
		log.Fatal(err)
	}
	sess := session.Must(session.NewSession(&aws.Config{Region: &region}))
	n, err := node.New(ec2.New(sess), metadata)
	if err != nil {
		log.Fatal(err)
	}
	if err = setHostname(*n.PrivateDnsName); err != nil {
		log.Fatal(err)
	}
	cluster, err := eks.Cluster(eksSvc.New(sess), n.ClusterName())
	if err != nil {
		log.Fatal(err)
	}

	info := struct {
		Cluster *eksSvc.Cluster
		Node    *node.Node
	}{
		Cluster: cluster,
		Node:    n,
	}

	if err = writeConfig("/var/lib/kubelet/kubeconfig", kubeconfig, info); err != nil {
		log.Fatal(err)
	}
	if err = writeConfig("/lib/systemd/system/kubelet.service", kubeletService, info); err != nil {
		log.Fatal(err)
	}
	if err = writeCertificate("/etc/kubernetes/pki/ca.crt", *cluster.CertificateAuthority.Data); err != nil {
		log.Fatal(err)
	}
	if err = runCommand("systemctl", "daemon-reload"); err != nil {
		log.Fatal(err)
	}
	if err = runCommand("systemctl", "restart", "kubelet.service"); err != nil {
		log.Fatal(err)
	}
}
