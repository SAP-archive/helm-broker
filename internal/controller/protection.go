package controller

import "github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"

type protection struct{}

func (p *protection) removeFinalizer(slice []string) []string {
	if !p.hasFinalizer(slice) {
		return slice
	}
	newSlice := make([]string, 0)
	for _, item := range slice {
		if item == v1alpha1.FinalizerAddonsConfiguration {
			continue
		}
		newSlice = append(newSlice, item)
	}
	return newSlice
}

func (protection) hasFinalizer(slice []string) bool {
	for _, item := range slice {
		if item == v1alpha1.FinalizerAddonsConfiguration {
			return true
		}
	}
	return false
}

func (p *protection) addFinalizer(slice []string) []string {
	if p.hasFinalizer(slice) {
		return slice
	}
	return append(slice, v1alpha1.FinalizerAddonsConfiguration)
}
