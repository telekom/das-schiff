package util

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// hasFinalizer checks whether an api object has a finalizer.
func HasFinalizer(meta v1.Object, finalizer string) bool {
	for _, f := range meta.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}
