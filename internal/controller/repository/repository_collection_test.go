package repository

import (
	"testing"

	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRepositoryCollection_AddRepository(t *testing.T) {
	// Given
	trc := NewRepositoryCollection()

	// When
	trc.AddRepository(&Repository{})
	trc.AddRepository(&Repository{})

	// Then
	assert.Len(t, trc.Repositories, 2)
}

func TestRepositoryCollection_completeAddons(t *testing.T) {
	// Given
	trc := NewRepositoryCollection()

	// When
	trc.AddRepository(
		&Repository{
			Addons: []*Entry{
				{ID: "84e70958-5ae1-49b7-a78c-25983d1b3d0e"},
				{ID: ""},
				{ID: "2285fb92-3eb1-4e93-bc47-eacd40344c90"},
			},
		})
	trc.AddRepository(
		&Repository{
			Addons: []*Entry{
				{ID: "e89b4535-1728-4577-a6f6-e67998733a0f"},
				{ID: "ceabec68-30cf-40fc-b2d9-0d4cd24aee45"},
				{ID: ""},
			},
		})

	// Then
	assert.Len(t, trc.completeAddons(), 4)
}

func TestRepositoryCollection_ReadyAddons(t *testing.T) {
	// Given
	trc := NewRepositoryCollection()

	// When
	trc.AddRepository(
		&Repository{
			Addons: []*Entry{
				{
					ID:    "84e70958-5ae1-49b7-a78c-25983d1b3d0e",
					Entry: v1alpha1.Addon{Status: v1alpha1.AddonStatusReady},
				},
				{
					ID:    "2285fb92-3eb1-4e93-bc47-eacd40344c90",
					Entry: v1alpha1.Addon{Status: v1alpha1.AddonStatusReady},
				},
				{
					ID:    "e89b4535-1728-4577-a6f6-e67998733a0f",
					Entry: v1alpha1.Addon{Status: v1alpha1.AddonStatusFailed},
				},
				{
					ID:    "ceabec68-30cf-40fc-b2d9-0d4cd24aee45",
					Entry: v1alpha1.Addon{Status: v1alpha1.AddonStatusReady},
				},
			},
		})

	// Then
	assert.Len(t, trc.ReadyAddons(), 3)
}

func TestRepositoryCollection_IsRepositoriesFailed(t *testing.T) {
	// Given
	trc := NewRepositoryCollection()

	// When
	trc.AddRepository(
		&Repository{
			Repository: v1alpha1.StatusRepository{Status: v1alpha1.RepositoryStatusReady},
		})
	trc.AddRepository(
		&Repository{
			Repository: v1alpha1.StatusRepository{Status: v1alpha1.RepositoryStatusReady},
		})

	// Then
	assert.False(t, trc.IsRepositoriesFailed())

	// When
	trc.AddRepository(&Repository{
		Addons: []*Entry{
			{
				Entry: v1alpha1.Addon{
					Status: v1alpha1.AddonStatusFailed,
				},
			},
		},
	})

	// Then
	assert.True(t, trc.IsRepositoriesFailed())
}

func TestRepositoryCollection_ReviseAddonDuplicationInRepository(t *testing.T) {
	// Given
	trc := NewRepositoryCollection()

	// When
	trc.AddRepository(
		&Repository{
			Addons: []*Entry{
				{
					ID:  "84e70958-5ae1-49b7-a78c-25983d1b3d0e",
					URL: "http://example.com/index.yaml",
					Entry: v1alpha1.Addon{
						Name:    "test",
						Version: "0.1",
						Status:  v1alpha1.AddonStatusReady,
					},
				},
				{
					ID:  "2285fb92-3eb1-4e93-bc47-eacd40344c90",
					URL: "http://example.com/index.yaml",
					Entry: v1alpha1.Addon{
						Name:    "test",
						Version: "0.2",
						Status:  v1alpha1.AddonStatusReady,
					},
				},
			},
		})
	trc.AddRepository(
		&Repository{
			Addons: []*Entry{
				{
					ID:  "e89b4535-1728-4577-a6f6-e67998733a0f",
					URL: "http://example.com/index-duplication.yaml",
					Entry: v1alpha1.Addon{
						Name:    "test",
						Version: "0.3",
						Status:  v1alpha1.AddonStatusReady,
					},
				},
				{
					ID:  "2285fb92-3eb1-4e93-bc47-eacd40344c90",
					URL: "http://example.com/index-duplication.yaml",
					Entry: v1alpha1.Addon{
						Name:    "test",
						Version: "0.4",
						Status:  v1alpha1.AddonStatusReady,
					},
				},
			},
		})
	trc.ReviseAddonDuplicationInRepository()

	// Then
	assert.Equal(t, string(v1alpha1.AddonStatusReady), string(findAddon(trc, "test", "0.1").Entry.Status))
	assert.Equal(t, string(v1alpha1.AddonStatusReady), string(findAddon(trc, "test", "0.2").Entry.Status))
	assert.Equal(t, string(v1alpha1.AddonStatusReady), string(findAddon(trc, "test", "0.3").Entry.Status))
	assert.Equal(t, string(v1alpha1.AddonStatusFailed), string(findAddon(trc, "test", "0.4").Entry.Status))
	assert.Equal(t,
		string(v1alpha1.AddonConflictInSpecifiedRepositories),
		string(findAddon(trc, "test", "0.4").Entry.Reason))
	assert.Equal(t,
		"Specified repositories have addons with the same ID: [url: http://example.com/index.yaml, addons: test:0.2]",
		string(findAddon(trc, "test", "0.4").Entry.Message))
}

func TestRepositoryCollection_ReviseAddonDuplicationInStorage(t *testing.T) {
	// Given
	trc := NewRepositoryCollection()
	list := &v1alpha1.AddonsConfigurationList{
		Items: []v1alpha1.AddonsConfiguration{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "addon-testing",
				},
				Status: v1alpha1.AddonsConfigurationStatus{
					CommonAddonsConfigurationStatus: v1alpha1.CommonAddonsConfigurationStatus{
						Repositories: []v1alpha1.StatusRepository{
							{
								URL: "http://example.com/index.yaml",
								Addons: []v1alpha1.Addon{
									{
										Name:    "test",
										Version: "0.2",
										Status:  v1alpha1.AddonStatusReady,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// When
	trc.AddRepository(
		&Repository{
			Addons: []*Entry{
				{
					ID:  "84e70958-5ae1-49b7-a78c-25983d1b3d0e",
					URL: "http://example.com/index.yaml",
					Entry: v1alpha1.Addon{
						Name:    "test",
						Version: "0.1",
						Status:  v1alpha1.AddonStatusReady,
					},
				},
				{
					ID:  "2285fb92-3eb1-4e93-bc47-eacd40344c90",
					URL: "http://example.com/index.yaml",
					Entry: v1alpha1.Addon{
						Name:    "test",
						Version: "0.2",
						Status:  v1alpha1.AddonStatusReady,
					},
				},
			},
		})
	trc.ReviseAddonDuplicationInStorage(list)

	// Then
	assert.Equal(t, string(v1alpha1.AddonStatusReady), string(findAddon(trc, "test", "0.1").Entry.Status))
	assert.Equal(t, string(v1alpha1.AddonStatusFailed), string(findAddon(trc, "test", "0.2").Entry.Status))
	assert.Equal(t,
		string(v1alpha1.AddonConflictWithAlreadyRegisteredAddons),
		string(findAddon(trc, "test", "0.2").Entry.Reason))
	assert.Equal(t,
		"An addon with the same ID is already registered: [ConfigurationName: addon-testing, url: http://example.com/index.yaml, addons: test:0.2]",
		string(findAddon(trc, "test", "0.2").Entry.Message))
}

func findAddon(rc *Collection, name, version string) *Entry {
	for _, addon := range rc.completeAddons() {
		if addon.Entry.Name == name && addon.Entry.Version == version {
			return addon
		}
	}

	return &Entry{}
}
