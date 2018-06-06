package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"text/template"
)

var sess = session.Must(session.NewSession())
var kubeconfig = template.Must(template.New("kubeconfig").Parse(`
apiVersion: v1
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
var kubeletService = template.Must(template.New("kubelet-service").Parse(`
[Unit]
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

type node struct {
	ClusterName     string
	Endpoint        string
	CertificateData string
	NodeIP          string
}

func main() {
	svc := eks.New(sess)
	input := &eks.ListClustersInput{}

	result, err := svc.ListClusters(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println(result)
}
