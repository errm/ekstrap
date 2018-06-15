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

var defaultTemplates = map[string]string{
	"/var/lib/kubelet/kubeconfig": `apiVersion: v1
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
        - "{{.Cluster.Name}}"`,

	"/lib/systemd/system/kubelet.service": `[Unit]
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
WantedBy=multi-user.target`,

	"/etc/kubernetes/pki/ca.crt": `{{.Cluster.CertificateAuthority.Data | b64dec}}`,
}
