package main

import (
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"text/template"

	"github.com/errm/ekstrap/pkg/eks"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	eksSvc "github.com/aws/aws-sdk-go/service/eks"
)

var metadata = ec2metadata.New(session.Must(session.NewSession()))

var kubeconfig = template.Must(template.New("kubeconfig").Parse(
	`apiVersion: v1
kind: Config
clusters:
- name: {{.ClusterName}}
  cluster:
    server: {{.Endpoint}}
    certificate-authority-data: {{.CertificateData}}
contexts:
- name: kubelet
  context:
    cluster: {{.ClusterName}}
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
        - "{{.ClusterName}}"
`))

var kubeletService = template.Must(template.New("kubelet-service").Parse(
	`[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/
After=docker.service
Requires=docker.service

[Service]
ExecStart=/usr/bin/kubelet   --address=0.0.0.0   --allow-privileged=true   --cloud-provider=aws   --cluster-dns=10.100.0.10   --cluster-domain=cluster.local   --cni-bin-dir=/opt/cni/bin   --cni-conf-dir=/etc/cni/net.d   --container-runtime=docker   --node-ip={{.NodeIP}}   --network-plugin=cni   --cgroup-driver=cgroupfs   --register-node=true   --kubeconfig=/var/lib/kubelet/kubeconfig   --feature-gates=RotateKubeletServerCertificate=true   --anonymous-auth=false   --client-ca-file=/etc/kubernetes/pki/ca.crt

Restart=on-failure
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`))

var sess = session.Must(session.NewSession(&aws.Config{
	Region: aws.String(Region()),
}))

var client = ec2.New(sess)
var id = InstanceId()

type node struct {
	ClusterName     string
	Endpoint        string
	CertificateData string
	NodeIP          string
}

func Region() string {
	result, e := metadata.GetMetadata("placement/availability-zone")
	if e != nil {
		log.Fatal(e)
	}
	return result[:len(result)-1]
}

func InstanceId() string {
	result, e := metadata.GetMetadata("instance-id")
	if e != nil {
		log.Fatal(e)
	}
	return result
}

func Instance() (*ec2.Instance, error) {
	output, err := client.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: []*string{&id}})
	if err != nil {
		return nil, err
	}
	return output.Reservations[0].Instances[0], nil
}

func EKSClusterName(instance *ec2.Instance) (string, error) {
	re := regexp.MustCompile(`kubernetes.io\/cluster\/([\w-]+)`)
	for _, t := range instance.Tags {
		if matches := re.FindStringSubmatch(*t.Key); len(matches) == 2 {
			return matches[1], nil
		}
	}
	return "", errors.New("kubernetes.io/cluster/<name> tag not set on instance")
}

func writeConfig(path string, templ *template.Template, data interface{}) error {
	directory := filepath.Dir(path)
	err := os.MkdirAll(directory, 0710)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0640)
	defer file.Close()
	if err != nil {
		return err
	}
	return templ.Execute(file, data)
}

func writeCertificate(path string, data string) error {
	directory := filepath.Dir(path)
	err := os.MkdirAll(directory, 0710)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0640)
	defer file.Close()
	if err != nil {
		return err
	}
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))
	_, err = io.Copy(file, decoder)
	return err
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
	file, err := os.Create("/etc/hostname")
	defer file.Close()
	if err != nil {
		return err
	}
	_, err = file.Write(h)
	if err != nil {
		return err
	}
	return syscall.Sethostname(h)
}

func main() {
	instance, err := Instance()
	if err != nil {
		log.Fatal(err)
	}
	if err = setHostname(*instance.PrivateDnsName); err != nil {
		log.Fatal(err)
	}
	ip := *instance.PrivateIpAddress
	name, err := EKSClusterName(instance)
	if err != nil {
		log.Fatal(err)
	}
	cluster, err := eks.Cluster(eksSvc.New(sess), name)
	if err != nil {
		log.Fatal(err)
	}
	info := node{
		ClusterName:     name,
		Endpoint:        *cluster.Endpoint,
		CertificateData: *cluster.CertificateAuthority.Data,
		NodeIP:          ip,
	}
	if err = writeConfig("/var/lib/kubelet/kubeconfig", kubeconfig, info); err != nil {
		log.Fatal(err)
	}
	if err = writeConfig("/lib/systemd/system/kubelet.service", kubeletService, info); err != nil {
		log.Fatal(err)
	}
	if err = writeCertificate("/etc/kubernetes/pki/ca.crt", info.CertificateData); err != nil {
		log.Fatal(err)
	}
	if err = runCommand("systemctl", "daemon-reload"); err != nil {
		log.Fatal(err)
	}
	if err = runCommand("systemctl", "restart", "kubelet.service"); err != nil {
		log.Fatal(err)
	}
}
