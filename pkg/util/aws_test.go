package util_test

import (
	"github.com/errm/ekstrap/pkg/util"
	"testing"
)

func TestIsAWSRegion(t *testing.T) {
	testCases := []struct {
		region string
		valid  bool
	}{
		{
			region: "us-east-2",
			valid:  true,
		},
		{
			region: "us-east-1",
			valid:  true,
		},
		{
			region: "us-west-1",
			valid:  true,
		},
		{
			region: "us-west-2",
			valid:  true,
		},
		{
			region: "ap-northeast-1",
			valid:  true,
		},
		{
			region: "cn-northwest-1",
			valid:  true,
		},
		{
			region: "us-gov-west-1",
			valid:  true,
		},
		{
			region: "sealand-central-1",
			valid:  false,
		},
		{
			region: "onion-ring-sandwich",
			valid:  false,
		},
		{
			region: "meh",
			valid:  false,
		},
	}
	for _, test := range testCases {
		result := util.IsAWSRegion(test.region)
		if result != test.valid {
			t.Errorf("Expected the result for: %v to be %v, but was %v", test.region, test.valid, result)
		}
	}
}
