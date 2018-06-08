package eks_test

import (
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

func TestClusterStatusActive(t *testing.T) {
	status := svc.ClusterStatusActive
	healthyCluster := &svc.Cluster{Status: &status}
	svc := &mockEKS{
		clusters: []*svc.Cluster{healthyCluster},
		errs:     []error{nil},
	}
	cluster, err := eks.Cluster(svc, "cluster-name")
	if err != nil {
		t.Error("there should be no error")
	}
	if cluster != healthyCluster {
		t.Error("the cluster was not returned correctly")
	}
}

func TestClusterStatusDeleting(t *testing.T) {
	status := svc.ClusterStatusDeleting
	deletingCluster := &svc.Cluster{Status: &status}
	svc := &mockEKS{
		clusters: []*svc.Cluster{deletingCluster},
		errs:     []error{nil},
	}
	cluster, err := eks.Cluster(svc, "cluster-name")
	if cluster != nil {
		t.Error("there should be no cluster returned")
	}
	expected := "Cannot use the EKS cluster: cluster-name, becuase it is DELETING"
	if err.Error() != expected {
		t.Errorf("Expected error test to be: %s, but was %s", expected, err.Error())
	}
}

func TestClusterStatusFailed(t *testing.T) {
	status := svc.ClusterStatusFailed
	failedCluster := &svc.Cluster{Status: &status}
	svc := &mockEKS{
		clusters: []*svc.Cluster{failedCluster},
		errs:     []error{nil},
	}
	cluster, err := eks.Cluster(svc, "cluster-name")
	if cluster != nil {
		t.Error("there should be no cluster returned")
	}
	expected := "Cannot use the EKS cluster: cluster-name, becuase it is FAILED"
	if err.Error() != expected {
		t.Errorf("Expected error test to be: %s, but was %s", expected, err.Error())
	}
}

func TestClusterStatusCreating(t *testing.T) {
	activeStatus := svc.ClusterStatusActive
	activeCluster := &svc.Cluster{Status: &activeStatus}

	creatingStatus := svc.ClusterStatusCreating
	creatingCluster := &svc.Cluster{Status: &creatingStatus}

	svc := &mockEKS{
		clusters: []*svc.Cluster{creatingCluster, activeCluster},
		errs:     []error{nil, nil},
	}

	cluster, err := eks.Cluster(svc, "cluster-name")
	if cluster != activeCluster {
		t.Error("it didn't return the correct cluster")
	}
	if err != nil {
		t.Error("there should be no error")
	}
}

func TestClusterNotFound(t *testing.T) {
	activeStatus := svc.ClusterStatusActive
	activeCluster := &svc.Cluster{Status: &activeStatus}

	notFoundError := awserr.New(svc.ErrCodeResourceNotFoundException, "Not found", nil)

	svc := &mockEKS{
		clusters: []*svc.Cluster{nil, activeCluster},
		errs:     []error{notFoundError, nil},
	}

	cluster, err := eks.Cluster(svc, "cluster-name")
	if cluster != activeCluster {
		t.Error("it didn't return the correct cluster")
	}
	if err != nil {
		t.Error("there should be no error")
	}
}

func TestClusterServiceError(t *testing.T) {
	activeStatus := svc.ClusterStatusActive
	activeCluster := &svc.Cluster{Status: &activeStatus}

	serviceError := awserr.New(svc.ErrCodeServiceUnavailableException, "AWS is broken", nil)

	svc := &mockEKS{
		clusters: []*svc.Cluster{nil, activeCluster},
		errs:     []error{serviceError, nil},
	}

	cluster, err := eks.Cluster(svc, "cluster-name")
	if cluster != activeCluster {
		t.Error("it didn't return the correct cluster")
	}
	if err != nil {
		t.Error("there should be no error")
	}
}
