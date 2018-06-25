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
	"testing"

	"github.com/errm/ekstrap/pkg/backoff"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

func init() {
	// An empty backoff just returns 0 all the time so the tests run fast
	b = backoff.Backoff{}
}

func TestNewNode(t *testing.T) {
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
	node, err := New(e, metadata)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if *node.InstanceId != "1234" {
		t.Error("Unexpected node returned")
	}

	if node.ClusterName() != "cluster-name" {
		t.Error("Expected returned node to have cluster-name")
	}
}

func TestNewErrors(t *testing.T) {
	metadataError := errors.New("error with metadata")
	ec2Error := errors.New("error with metadata")

	e := &mockEC2{err: ec2Error}
	metadata := mockMetadata{err: metadataError}

	_, err := New(e, metadata)
	if err != metadataError {
		t.Errorf("expected error: %s to be %s", err, metadataError)
	}

	metadata = mockMetadata{
		data: map[string]string{
			"instance-id": "1234",
		},
	}

	_, err = New(e, metadata)
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
		node, err := New(e, metadata)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if node.MaxPods != test.expected {
			t.Errorf("expected MaxPods for %v to be: %v, but it was %v", test.instanceType, test.expected, node.MaxPods)
		}
	}
}

func tag(key, value string) *ec2.Tag {
	return &ec2.Tag{
		Key:   &key,
		Value: &value,
	}
}

type mockEC2 struct {
	ec2iface.EC2API
	tags         [][]*ec2.Tag
	instanceType string
	err          error
}

func (m *mockEC2) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	var tags []*ec2.Tag
	//Pop the first set of tags
	tags, m.tags = m.tags[0], m.tags[1:]
	if len(input.InstanceIds) > 0 {
		return &ec2.DescribeInstancesOutput{
			Reservations: []*ec2.Reservation{{
				Instances: []*ec2.Instance{
					{
						InstanceId:   input.InstanceIds[0],
						Tags:         tags,
						InstanceType: &m.instanceType,
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
