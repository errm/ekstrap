package util

import "regexp"

// IsAWSRegion provides a sanity check that an AWS region seems to be correct.
// It is just a sanity check but does not guarantee that a region exists, just
// that the string is in roughly the correct format.
func IsAWSRegion(region string) bool {
	return regexp.MustCompile(`^[a-z\-]{2,6}-[a-z]{4,9}-\d$`).MatchString(region)
}
