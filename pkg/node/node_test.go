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

package node

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/errm/ekstrap/pkg/backoff"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

var use1 = "us-east-1"
var usw2 = "us-west-2"

func disableBackoff() {
	// An empty backoff just returns 0 all the time so the tests run fast
	b = backoff.Backoff{}
}

func TestNewNode(t *testing.T) {
	disableBackoff()
	e := &mockEC2{
		tags: [][]*ec2.Tag{
			{},
			{},
			{tag("kubernetes.io/cluster/cluster-name", "owned")},
		},
	}
	metadata := mockMetadata{
		data: map[string]string{
			"instance-id": "1234",
		},
	}
	node, err := New(e, metadata, &use1, "docker")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if *node.InstanceId != "1234" {
		t.Error("Unexpected node returned")
	}

	if node.ClusterName() != "cluster-name" {
		t.Error("Expected returned node to have cluster-name")
	}

	if node.Region != use1 {
		t.Errorf("Expected %s, to eq %s", node.Region, use1)
	}
}

func TestNodeLabels(t *testing.T) {
	disableBackoff()
	e := &mockEC2{
		tags: [][]*ec2.Tag{
			{},
			{},
			{
				tag("kubernetes.io/cluster/cluster-name", "owned"),
				tag("k8s.io/cluster-autoscaler/node-template/label/nvidia-gpu", "K80"),
			},
		},
	}
	metadata := mockMetadata{
		data: map[string]string{
			"instance-id": "1234",
		},
	}
	node, err := New(e, metadata, &use1, "docker")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	expected := []string{
		"node-role.kubernetes.io/worker=true",
		"nvidia-gpu=K80",
	}

	if !reflect.DeepEqual(node.Labels(), expected) {
		t.Errorf("Expected node.Labels to be %v but was %v", expected, node.Labels())
	}

	e = &mockEC2{
		tags: [][]*ec2.Tag{
			{},
			{},
			{
				tag("kubernetes.io/cluster/cluster-name", "owned"),
				tag("k8s.io/cluster-autoscaler/node-template/label/nvidia-gpu", "K80"),
			},
		},
		instanceLifecycle: ec2.InstanceLifecycleTypeSpot,
	}
	node, err = New(e, metadata, &use1, "docker")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	expected = []string{
		"node-role.kubernetes.io/spot-worker=true",
		"nvidia-gpu=K80",
	}

	if !reflect.DeepEqual(node.Labels(), expected) {
		t.Errorf("Expected node.Labels to be %v but was %v", expected, node.Labels())
	}
}

func TestNodeTaints(t *testing.T) {
	disableBackoff()
	e := &mockEC2{
		tags: [][]*ec2.Tag{
			{},
			{},
			{
				tag("kubernetes.io/cluster/cluster-name", "owned"),
				tag("k8s.io/cluster-autoscaler/node-template/label/foo", "bar"),
				tag("k8s.io/cluster-autoscaler/node-template/taint/dedicated", "foo:NoSchedule"),
				tag("k8s.io/cluster-autoscaler/node-template/label/nvidia-gpu", "K80"),
			},
		},
	}
	metadata := mockMetadata{
		data: map[string]string{
			"instance-id": "1234",
		},
	}
	node, err := New(e, metadata, &use1, "docker")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	expected := []string{
		"dedicated=foo:NoSchedule",
	}

	if !reflect.DeepEqual(node.Taints(), expected) {
		t.Errorf("Expected node.Taints to be %v but was %v", expected, node.Taints())
	}
}

func TestClusterDNS(t *testing.T) {
	e := &mockEC2{
		PrivateIPAddress: "10.1.123.4",
		tags: [][]*ec2.Tag{
			{tag("kubernetes.io/cluster/cluster-name", "owned")},
		},
	}
	node, err := New(e, mockMetadata{}, &use1, "docker")

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if node.ClusterDNS() != "172.20.0.10" {
		t.Errorf("expected ClusterDNS to be 172.20.0.10 got: %s", node.ClusterDNS())
	}

	e = &mockEC2{
		PrivateIPAddress: "172.16.45.45",
		tags: [][]*ec2.Tag{
			{tag("kubernetes.io/cluster/cluster-name", "owned")},
		},
	}
	node, err = New(e, mockMetadata{}, &use1, "docker")

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if node.ClusterDNS() != "10.100.0.10" {
		t.Errorf("expected ClusterDNS to be 10.100.0.10 got: %s", node.ClusterDNS())
	}
}

func TestNewErrors(t *testing.T) {
	metadataError := errors.New("error with metadata")
	ec2Error := errors.New("error with metadata")

	e := &mockEC2{err: ec2Error}
	metadata := mockMetadata{err: metadataError}

	_, err := New(e, metadata, &use1, "docker")
	if err != metadataError {
		t.Errorf("expected error: %s to be %s", err, metadataError)
	}

	metadata = mockMetadata{
		data: map[string]string{
			"instance-id": "1234",
		},
	}

	_, err = New(e, metadata, &use1, "docker")
	if err != ec2Error {
		t.Errorf("expected error: %s to be %s", err, ec2Error)
	}
}

func TestClusterName(t *testing.T) {
	tests := []struct {
		node     Node
		expected string
	}{
		{

			node:     Node{Instance: &ec2.Instance{Tags: []*ec2.Tag{tag("kubernetes.io/cluster/this-is-a_name", "owned")}}},
			expected: "this-is-a_name",
		},
		{

			node:     Node{Instance: &ec2.Instance{Tags: []*ec2.Tag{tag("kubernetes.io/cluster/some-other-name", "owned")}}},
			expected: "some-other-name",
		},
		{

			node:     Node{Instance: &ec2.Instance{Tags: []*ec2.Tag{tag("kubernetes.io/cluster/this-is-a_name", "owned"), tag("unrelated", "tag")}}},
			expected: "this-is-a_name",
		},
		{
			node:     Node{Instance: &ec2.Instance{}},
			expected: "",
		},
		{

			node:     Node{Instance: &ec2.Instance{Tags: []*ec2.Tag{tag("unrelated", "tag")}}},
			expected: "",
		},
	}

	for _, test := range tests {
		actual := test.node.ClusterName()
		if actual != test.expected {
			t.Errorf("expected: %s to equal %s", actual, test.expected)
		}
	}
}

func TestPauseImage(t *testing.T) {
	arm := "arm64"
	amd := "x86_64"
	tests := []struct {
		node     Node
		expected string
	}{
		{
			node:     Node{Region: "us-east-1", Instance: &ec2.Instance{Architecture: &amd}},
			expected: "602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause-amd64:3.1",
		},
		{
			node:     Node{Region: "eu-west-1", Instance: &ec2.Instance{}},
			expected: "602401143452.dkr.ecr.eu-west-1.amazonaws.com/eks/pause-amd64:3.1",
		},
		{
			node:     Node{Region: "ap-east-1", Instance: &ec2.Instance{Architecture: &amd}},
			expected: "800184023465.dkr.ecr.ap-east-1.amazonaws.com/eks/pause-amd64:3.1",
		},
		{
			node:     Node{Region: "me-south-1", Instance: &ec2.Instance{Architecture: &amd}},
			expected: "558608220178.dkr.ecr.me-south-1.amazonaws.com/eks/pause-amd64:3.1",
		},
		{
			node:     Node{Region: "us-east-1", Instance: &ec2.Instance{Architecture: &arm}},
			expected: "602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause-arm64:3.1",
		},
	}

	for _, test := range tests {
		actual := test.node.PauseImage()
		if actual != test.expected {
			t.Errorf("expected: %s to equal %s", actual, test.expected)
		}
	}
}

func TestMaxPods(t *testing.T) {
	tests := []struct {
		instanceType string
		expected     int
	}{
		{

			instanceType: "c4.large",
			expected:     27,
		},
		{

			instanceType: "x1.16xlarge",
			expected:     232,
		},
		{

			instanceType: "t2.medium",
			expected:     15,
		},
		{

			instanceType: "unknown.instance",
			expected:     0,
		},
	}

	for _, test := range tests {
		e := &mockEC2{
			tags: [][]*ec2.Tag{
				{tag("kubernetes.io/cluster/cluster-name", "owned")},
			},
			instanceType: test.instanceType,
		}
		metadata := mockMetadata{
			data: map[string]string{
				"instance-id": "1234",
			},
		}
		node, err := New(e, metadata, &usw2, "docker")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if node.MaxPods() != test.expected {
			t.Errorf("expected MaxPods for %v to be: %v, but it was %v", test.instanceType, test.expected, node.MaxPods())
		}
	}
}

func TestReservedCPU(t *testing.T) {
	tests := []struct {
		instanceType string
		expected     string
	}{
		{

			instanceType: "m3.medium",
			expected:     "60m",
		},
		{

			instanceType: "m5.large",
			expected:     "70m",
		},
		{

			instanceType: "c5.xlarge",
			expected:     "80m",
		},
		{

			instanceType: "c5.2xlarge",
			expected:     "90m",
		},
		{

			instanceType: "h1.4xlarge",
			expected:     "110m",
		},
		{

			instanceType: "i3.8xlarge",
			expected:     "150m",
		},
		{

			instanceType: "m5.24xlarge",
			expected:     "310m",
		},
		{

			instanceType: "x1e.32xlarge",
			expected:     "390m",
		},
		{

			instanceType: "unexpected.instance",
			expected:     "",
		},
	}

	for _, test := range tests {
		e := &mockEC2{
			tags: [][]*ec2.Tag{
				{tag("kubernetes.io/cluster/cluster-name", "owned")},
			},
			instanceType: test.instanceType,
		}
		metadata := mockMetadata{
			data: map[string]string{
				"instance-id": "1234",
			},
		}
		node, err := New(e, metadata, &usw2, "docker")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if node.ReservedCPU() != test.expected {
			t.Errorf("expected ReservedCPU for %v to be: %v, but it was %v", test.instanceType, test.expected, node.ReservedCPU())
		}
	}
}

func TestMemory(t *testing.T) {
	tests := []struct {
		instanceType string
		expected     string
	}{
		{

			instanceType: "m3.medium",
			expected:     "960Mi",
		},
		{

			instanceType: "m5.large",
			expected:     "1843Mi",
		},
		{

			instanceType: "c5.2xlarge",
			expected:     "2662Mi",
		},
		{

			instanceType: "h1.4xlarge",
			expected:     "5612Mi",
		},
		{

			instanceType: "i3.8xlarge",
			expected:     "11919Mi",
		},
		{

			instanceType: "m5.24xlarge",
			expected:     "14787Mi",
		},
		{

			instanceType: "x1e.32xlarge",
			expected:     "86876Mi",
		},
		{

			instanceType: "unexpected.instance",
			expected:     "",
		},
	}

	for _, test := range tests {
		e := &mockEC2{
			tags: [][]*ec2.Tag{
				{tag("kubernetes.io/cluster/cluster-name", "owned")},
			},
			instanceType: test.instanceType,
		}
		metadata := mockMetadata{
			data: map[string]string{
				"instance-id": "1234",
			},
		}
		node, err := New(e, metadata, &usw2, "docker")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if node.ReservedMemory() != test.expected {
			t.Errorf("expected ReservedMemory for %v to be: %v, but it was %v", test.instanceType, test.expected, node.ReservedMemory())
		}
	}
}

func TestInstanceTypeInfo(t *testing.T) {
	expected := keys(InstanceENIsAvailable)
	tests := []struct {
		dataset string
		keys    []string
	}{
		{
			dataset: "InstanceENIsAvailable",
			keys:    keys(InstanceENIsAvailable),
		},
		{
			dataset: "InstanceIPsAvailable",
			keys:    keys(InstanceIPsAvailable),
		},
		{
			dataset: "InstanceCores",
			keys:    keys(InstanceCores),
		},
		{
			dataset: "InstanceMemory",
			keys:    keys(InstanceMemory),
		},
	}

	for _, test := range tests {
		if !reflect.DeepEqual(test.keys, expected) {
			t.Errorf("expected %v, had diff %#v", test.dataset, diference(test.keys, expected))
		}
	}
}

func keys(m map[string]int) (ks []string) {
	for key := range m {
		ks = append(ks, key)
	}
	sort.Strings(ks)
	return
}

func diference(a, b []string) (d []string) {
	d = append(d, diff(a, b)...)
	d = append(d, diff(b, a)...)
	return
}

func diff(a, b []string) (d []string) {
	m := make(map[string]bool)
	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			d = append(d, item)
		}
	}
	return
}

func tag(key, value string) *ec2.Tag {
	return &ec2.Tag{
		Key:   &key,
		Value: &value,
	}
}

type mockEC2 struct {
	PrivateIPAddress string
	ec2iface.EC2API
	tags              [][]*ec2.Tag
	instanceType      string
	instanceLifecycle string
	err               error
}

func (m *mockEC2) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	var tags []*ec2.Tag
	//Pop the first set of tags
	tags, m.tags = m.tags[0], m.tags[1:]

	var instanceLifecycle *string
	if m.instanceLifecycle != "" {
		instanceLifecycle = &m.instanceLifecycle
	}
	if len(input.InstanceIds) > 0 {
		return &ec2.DescribeInstancesOutput{
			Reservations: []*ec2.Reservation{{
				Instances: []*ec2.Instance{
					{
						InstanceId:        input.InstanceIds[0],
						Tags:              tags,
						InstanceType:      &m.instanceType,
						PrivateIpAddress:  &m.PrivateIPAddress,
						InstanceLifecycle: instanceLifecycle,
					},
				},
			},
			},
		}, nil
	}
	return nil, nil
}

type mockMetadata struct {
	data map[string]string
	err  error
}

func (m mockMetadata) GetMetadata(key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.data[key], nil
}
