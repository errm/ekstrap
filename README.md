# ekstrap

[![deb package](https://img.shields.io/badge/deb-packagecloud.io-844fec.svg)](https://packagecloud.io/errm/ekstrap) [![rpm package](https://img.shields.io/badge/rpm-packagecloud.io-844fec.svg)](https://packagecloud.io/errm/ekstrap) [![Build Status](https://travis-ci.org/errm/ekstrap.svg?branch=master)](https://travis-ci.org/errm/ekstrap) [![Go Report Card](https://goreportcard.com/badge/github.com/errm/ekstrap)](https://goreportcard.com/report/github.com/errm/ekstrap)

ekstrap is a simple tool to bootstrap the configuration on Kuberntes nodes so that they may join an [EKS](https://aws.amazon.com/eks/) cluster.

## Usage

When run on an ec2 node ekstrap performs several tasks.

* Discovers the name of your EKS cluster by looking for the `kubernetes.io/cluster/<name>` tag.
* Discovers the endpoint and CA certificate of your EKS cluster.
* Updates the hostname of the node to match the `PrivateDnsName` from the EC2 API.
* Writes a kubeconfig file configured to connect to your EKS cluster to `/var/lib/kubelet/kubeconfig`.
* Writes a systemd unit file to `/lib/systemd/system/kubelet.service`.
* Writes the cluster CA certificate to `/etc/kubernetes/pki/ca.crt`.
* Restarts the kubelet unit.


In order to run ekstrap your instance should have an IAM instance profile that allows the `EC2::DescribeInstances` action and the `EKS::DescribeCluster` action. Both of these actions are already included in the AWS managed policy `arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy` along with the other permissions that the kubelet requires to connect to your cluster, it is recommended therefore to simply attach this policy to your instance role/profile.

## Installation

The latest version of ekstrap can be downloaded from [github](https://github.com/errm/ekstrap/releases)

```
$ curl -LO https://github.com/errm/ekstrap/releases/download/v0.0.2/ekstrap_0.0.2_linux_x86_64
```

You should check that the provided checksums match before you use the binary.

```
$ curl -LO https://github.com/errm/ekstrap/releases/download/v0.0.2/ekstrap_0.0.2_checksums.txt
$ sha256sum -c ekstrap_0.0.2_checksums.txt
ekstrap_0.0.2_linux_x86_64: OK
```

Install the ekstrap binary into a suitable location e.g. `/usr/local/bin/ekstrap`

```
$ install -m755 ekstrap_0.0.2_linux_x86_64 /usr/local/bin/ekstrap
```

You might choose to run ekstrap from a userdata script:

```bash
#!/bin/bash

ekstrap
```

Or with a [oneshot unit](example/ekstrap.service)

```systemd
[Unit]
Description=Configures Kubernetes EKS Worker Node
Before=kubelet.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/ekstrap
RemainAfterExit=true

[Install]
WantedBy=multi-user.target
```

Remember that because ekstrap writes config files with strict permissions and interacts with the init system, it needs to run as root.

## Development

`make`

Will run the tests and build a binary

### Linting

On CI we run [some](.gometalinter.json) linting processes.

To run localy:

* First install gometalinter: `make install-linter`.
* Then run: `make lint`

## Dependencies

To build ekstrap you need [go](https://golang.org/)

Dependencies are checked into the vendor folder so you can build the project without any extra tools,
but if you need to change or update them you will need to install [dep](https://golang.github.io/dep/).

If you want a tiny binary, install [upx](https://upx.github.io/) and run the `make compress` task.

ekstrap currently only works with systemd, if you want us to support another init system please comment here https://github.com/errm/ekstrap/issues/28.

## Contributing

If you want to contribute to this tool:

* Thank You!
* Open an issue
* Or a PR
* Try to write tests if you are adding code / features

## Thanks

<a href="https://packagecloud.io/"><img height="46" width="158" alt="Private NPM registry and Maven, RPM, DEB, PyPi and RubyGem Repository · packagecloud" src="https://packagecloud.io/images/packagecloud-badge.png" /></a>

## License

Apache 2.0
