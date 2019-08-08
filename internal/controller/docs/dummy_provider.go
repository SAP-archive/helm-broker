package docs

import "github.com/kyma-project/helm-broker/internal"

// DummyProvider is an implementation which does not perform any work but have the same interface as the Provider
type DummyProvider struct {
}

// EnsureDocsTopic fulfills the docsFacade interface
func (s *DummyProvider) EnsureDocsTopic(addon *internal.Addon, namespace string) error {
	return nil
}

// EnsureDocsTopicRemoved fulfills the docsFacade interface
func (*DummyProvider) EnsureDocsTopicRemoved(id string, namespace string) error {
	return nil
}

// EnsureClusterDocsTopic fulfills the docsFacade interface
func (*DummyProvider) EnsureClusterDocsTopic(addon *internal.Addon) error {
	return nil
}

// EnsureClusterDocsTopicRemoved fulfills the docsFacade interface
func (*DummyProvider) EnsureClusterDocsTopicRemoved(id string) error {
	return nil
}
