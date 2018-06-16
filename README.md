# ekstrap

ekstrap is a simple tool to bootstrap the configuration on Kuberntes nodes so that they may join an [EKS](https://aws.amazon.com/eks/) cluster.

## Usage

When run on an ec2 node ekstrap performs several tasks.

* Discovers the name of your EKS cluster by looking for the `kubernetes.io/cluster/<name>` tag.
* Discovers the enpoint and CA certificate of your EKS cluster.
* Updates the hostname of the node to match the `PrivateDnsName` from the ec2 api.
* Writes a kubeconfig file configured to connect to your EKS cluster to `/var/lib/kubelet/kubeconfig`.
* Writes a systemd unit file to `/lib/systemd/system/kubelet.service`
* Writes the cluster CA certificate to `/etc/kubernetes/pki/ca.crt`
* Restarts the kubelet unit

You might choose to run ekstrap from a userdata script, or with a oneshot unit.

In order to run ekstrap your instance should have an IAM instance profile that allows the `EC2::DescribeInstances` action and the `EKS::DescribeCluster` action. Both of these actions are allready included in the AWS managed policy `arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy` along with the other permissions that the kubelet requires to connect to your cluster.

## Development

`make`

Will run the tests and build a binary

## Dependencies

To build ekstrap you need [go](https://golang.org/)

Dependencies are checked into the vendor folder so you can build the project without any extra tools,
but if you need to change or update them you will need to install [dep](https://golang.github.io/dep/).

If you want a tiny binary, install [upx](https://upx.github.io/) and run the `make compress` task.

## License

Apache 2.0
