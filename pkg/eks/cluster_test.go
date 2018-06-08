package eks_test

import (
	"errors"
	"github.com/errm/ekstrap/pkg/eks"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	svc "github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

type mockEKS struct {
	eksiface.EKSAPI
	clusters []*svc.Cluster
	errs     []error
}

func (m *mockEKS) DescribeCluster(input *svc.DescribeClusterInput) (*svc.DescribeClusterOutput, error) {
	var cluster *svc.Cluster
	// Pop last cluster from clusters
	cluster, m.clusters = m.clusters[0], m.clusters[1:]
	output := &svc.DescribeClusterOutput{
		Cluster: cluster,
	}
	var err error
	// Pop last error from errs
	err, m.errs = m.errs[0], m.errs[1:]
	return output, err
}

var activeStatus = svc.ClusterStatusActive
var activeCluster = &svc.Cluster{Status: &activeStatus}
var deletingStatus = svc.ClusterStatusDeleting
var deletingCluster = &svc.Cluster{Status: &deletingStatus}
var failedStatus = svc.ClusterStatusFailed
var failedCluster = &svc.Cluster{Status: &failedStatus}
var creatingStatus = svc.ClusterStatusCreating
var creatingCluster = &svc.Cluster{Status: &creatingStatus}
var notFoundError = awserr.New(svc.ErrCodeResourceNotFoundException, "Not found", nil)
var serviceError = awserr.New(svc.ErrCodeServiceUnavailableException, "AWS is broken", nil)

func TestCluster(t *testing.T) {
	tests := []struct {
		clusters      []*svc.Cluster
		errors        []error
		expected      *svc.Cluster
		expectedError error
	}{
		{
			clusters:      []*svc.Cluster{activeCluster},
			errors:        []error{nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*svc.Cluster{deletingCluster},
			errors:        []error{nil},
			expected:      nil,
			expectedError: errors.New("Cannot use the EKS cluster: cluster-name, becuase it is DELETING"),
		},
		{
			clusters:      []*svc.Cluster{failedCluster},
			errors:        []error{nil},
			expected:      nil,
			expectedError: errors.New("Cannot use the EKS cluster: cluster-name, becuase it is FAILED"),
		},
		{
			clusters:      []*svc.Cluster{creatingCluster, activeCluster},
			errors:        []error{nil, nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*svc.Cluster{nil, activeCluster},
			errors:        []error{notFoundError, nil},
			expected:      activeCluster,
			expectedError: nil,
		},
		{
			clusters:      []*svc.Cluster{nil, activeCluster},
			errors:        []error{serviceError, nil},
			expected:      activeCluster,
			expectedError: nil,
		},
	}

	for _, test := range tests {
		svc := &mockEKS{
			clusters: test.clusters,
			errs:     test.errors,
		}
		cluster, err := eks.Cluster(svc, "cluster-name")
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
