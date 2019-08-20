package repository

import (
	"fmt"
	"testing"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryEntry_IsReady(t *testing.T) {
	// Given
	ta := testAddon()

	// Then
	assert.Equal(t, internal.AddonName("default-addon"), ta.AddonWithCharts.Addon.Name)
	assert.Equal(t, "1.0", ta.AddonWithCharts.Addon.Version.Original())
	assert.True(t, ta.IsReady())
}

func TestRepositoryEntry_IsComplete(t *testing.T) {
	// Given
	ta := testAddon()

	// When
	ta.ID = "7929c146-bf8d-4b65-8eba-8348ac956546"

	// Then
	assert.True(t, ta.IsComplete())
}

func TestRepositoryEntry_FetchingError(t *testing.T) {
	// Given
	ta := testAddon()

	// When
	ta.FetchingError(fmt.Errorf("some error:a:b:c:d:e:f:g"))

	// Then
	assert.False(t, ta.IsReady())
	assert.Equal(t, v1alpha1.AddonStatusFailed, ta.AddonWithCharts.Addon.Status)
	assert.Equal(t, v1alpha1.AddonFetchingError, ta.AddonWithCharts.Addon.Reason)
	assert.Equal(t, "Fetching failed due to error: 'some error:a:b:c'", ta.AddonWithCharts.Addon.Message)
}

func TestRepositoryEntry_LoadingError(t *testing.T) {
	// Given
	ta := testAddon()

	// When
	ta.LoadingError(fmt.Errorf("loading error"))

	// Then
	assert.False(t, ta.IsReady())
	assert.Equal(t, v1alpha1.AddonStatusFailed, ta.AddonWithCharts.Addon.Status)
	assert.Equal(t, v1alpha1.AddonLoadingError, ta.AddonWithCharts.Addon.Reason)
	assert.Equal(t, "Loading failed due to error: 'loading error'", ta.AddonWithCharts.Addon.Message)
}

func TestRepositoryEntry_ConflictInSpecifiedRepositories(t *testing.T) {
	// Given
	ta := testAddon()

	// When
	ta.ConflictInSpecifiedRepositories(fmt.Errorf("id exist in repositories"))

	// Then
	assert.False(t, ta.IsReady())
	assert.Equal(t, v1alpha1.AddonStatusFailed, ta.AddonWithCharts.Addon.Status)
	assert.Equal(t, v1alpha1.AddonConflictInSpecifiedRepositories, ta.AddonWithCharts.Addon.Reason)
	assert.Equal(t, "Specified repositories have addons with the same ID: id exist in repositories", ta.AddonWithCharts.Addon.Message)
}

func TestRepositoryEntry_ConflictWithAlreadyRegisteredAddons(t *testing.T) {
	// Given
	ta := testAddon()

	// When
	ta.ConflictWithAlreadyRegisteredAddons(fmt.Errorf("id exist in storage"))

	// Then
	assert.False(t, ta.IsReady())
	assert.Equal(t, v1alpha1.AddonStatusFailed, ta.AddonWithCharts.Addon.Status)
	assert.Equal(t, v1alpha1.AddonConflictWithAlreadyRegisteredAddons, ta.AddonWithCharts.Addon.Reason)
	assert.Equal(t, "An addon with the same ID is already registered: id exist in storage", ta.AddonWithCharts.Addon.Message)
}

func TestRepositoryEntry_RegisteringError(t *testing.T) {
	// Given
	ta := testAddon()

	// When
	ta.RegisteringError(fmt.Errorf("cannot register"))

	// Then
	assert.False(t, ta.IsReady())
	assert.Equal(t, v1alpha1.AddonStatusFailed, ta.AddonWithCharts.Addon.Status)
	assert.Equal(t, v1alpha1.AddonRegisteringError, ta.AddonWithCharts.Addon.Reason)
	assert.Equal(t, "Registering failed due to error: 'cannot register'", ta.AddonWithCharts.Addon.Message)
}

func testAddon() *Entry {
	return NewRepositoryEntry("default-addon", "1.0", "https://example.com")
}
