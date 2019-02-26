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

package eks

import (
	"errors"
	"testing"

	"github.com/errm/ekstrap/pkg/backoff"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

func disableBackoff() {
	// An empty backoff just returns 0 all the time so the tests run fast
	b = backoff.Backoff{}
}

func TestCluster(t *testing.T) {
	disableBackoff()
	activeStatus := eks.ClusterStatusActive
	activeCluster := &eks.Cluster{Status: &activeStatus}
	deletingStatus := eks.ClusterStatusDeleting
	deletingCluster := &eks.Cluster{Status: &deletingStatus}
	failedStatus := eks.ClusterStatusFailed
	failedCluster := &eks.Cluster{Status: &failedStatus}
	creatingStatus := eks.ClusterStatusCreating
	creatingCluster := &eks.Cluster{Status: &creatingStatus}
	notFoundError := awserr.New(eks.ErrCodeResourceNotFoundException, "Not found", nil)
	serviceError := awserr.New(eks.ErrCodeServiceUnavailableException, "AWS is broken", nil)
	clientError := awserr.New(eks.ErrCodeClientException, "Your credentials are no good", nil)
	tests := []struct {
		clusters      []*eks.Cluster
		errors        []error
		expected      *eks.Cluster
		expectedError error
	}{
		{
			clusters:      []*eks.Cluster{activeCluster},
			errors:        []error{nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*eks.Cluster{deletingCluster},
			errors:        []error{nil},
			expected:      nil,
			expectedError: errors.New("cannot use the EKS cluster: cluster-name, because it is DELETING"),
		},
		{
			clusters:      []*eks.Cluster{failedCluster},
			errors:        []error{nil},
			expected:      nil,
			expectedError: errors.New("cannot use the EKS cluster: cluster-name, because it is FAILED"),
		},
		{
			clusters:      []*eks.Cluster{creatingCluster, activeCluster},
			errors:        []error{nil, nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*eks.Cluster{nil, activeCluster},
			errors:        []error{notFoundError, nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*eks.Cluster{nil, activeCluster},
			errors:        []error{serviceError, nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*eks.Cluster{nil, nil, nil, creatingCluster, activeCluster},
			errors:        []error{notFoundError, serviceError, notFoundError, nil, nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*eks.Cluster{nil},
			errors:        []error{clientError},
			expected:      nil,
			expectedError: clientError,
		},
	}

	for _, test := range tests {
		svc := &mockEKS{
			clusters: test.clusters,
			errs:     test.errors,
		}
		cluster, err := Cluster(svc, "cluster-name")
		if cluster != test.expected {
			t.Errorf("expected cluster: %v, got %v", test.expected, cluster)
		}
		if test.expectedError == nil {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		} else {
			if err.Error() != test.expectedError.Error() {
				t.Errorf("expected error message: %s, got %s", test.expectedError.Error(), err.Error())
			}
		}
	}
}

type mockEKS struct {
	eksiface.EKSAPI
	clusters []*eks.Cluster
	errs     []error
}

func (m *mockEKS) DescribeCluster(input *eks.DescribeClusterInput) (*eks.DescribeClusterOutput, error) {
	var cluster *eks.Cluster
	// Pop first cluster from clusters
	cluster, m.clusters = m.clusters[0], m.clusters[1:]
	output := &eks.DescribeClusterOutput{
		Cluster: cluster,
	}
	var err error
	// Pop last error from errs
	err, m.errs = m.errs[0], m.errs[1:]
	return output, err
}
