package internal_test

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ghodss/yaml"
	"github.com/kyma-project/helm-broker/internal"
)

func TestIndexDTO(t *testing.T) {
	// GIVEN
	data := `
apiVersion: v1
entries:
  redis:
    - name: redis
      description: Redis service
      version: 0.0.1
`
	dto := internal.Index{}
	// WHEN
	err := yaml.Unmarshal([]byte(data), &dto)

	// THEN
	require.NoError(t, err)
	require.Len(t, dto.Entries, 1)
	redis, ex := dto.Entries["redis"]
	assert.True(t, ex)
	assert.Len(t, redis, 1)
	v001 := redis[0]
	assert.Equal(t, "redis", v001.DisplayName)
	assert.Equal(t, internal.AddonVersion("0.0.1"), v001.Version)
	assert.Equal(t, "Redis service", v001.Description)

}

func TestChartRefGobEncodeDecode(t *testing.T) {
	for sym, exp := range map[string]internal.ChartRef{
		"A":          {Name: "NameA", Version: *semver.MustParse("0.0.1")},
		"empty/name": {Name: "NameA"},
		"empty/all":  {},
	} {
		t.Run(sym, func(t *testing.T) {
			// GIVEN:
			buf := bytes.Buffer{}
			enc := gob.NewEncoder(&buf)
			dec := gob.NewDecoder(&buf)
			var got internal.ChartRef

			// WHEN:
			err := enc.Encode(&exp)
			require.NoError(t, err)

			err = dec.Decode(&got)
			require.NoError(t, err)

			// THEN:
			assert.Equal(t, exp.Name, got.Name)
			assert.Equal(t, exp.Version.String(), got.Version.String())
		})
	}
}

func TestCanBeProvision(t *testing.T) {
	// Given
	namespace := internal.Namespace("test-addon-namespace")
	collection := []*internal.Instance{
		{ServiceID: "a1", Namespace: "test-addon-namespace"},
		{ServiceID: "a2", Namespace: "test-addon-namespace"},
		{ServiceID: "a3", Namespace: "test-addon-namespace"},
		{ServiceID: "a2", Namespace: "other-addon-namespace"},
	}

	addonExist := internal.Addon{
		Metadata: internal.AddonMetadata{
			ProvisionOnlyOnce: true,
		},
		ID: "a1",
	}
	addonNotExist := internal.Addon{
		Metadata: internal.AddonMetadata{
			ProvisionOnlyOnce: true,
		},
		ID: "a5",
	}
	addonManyProvision := internal.Addon{
		Metadata: internal.AddonMetadata{
			ProvisionOnlyOnce: false,
		},
		ID: "a1",
	}

	// WHEN/THEN
	assert.False(t, addonExist.IsProvisioningAllowed(namespace, collection))
	assert.True(t, addonExist.IsProvisioningAllowed("other-addon-namespace", collection))
	assert.True(t, addonExist.IsProvisioningAllowed("other-ns", collection))
	assert.True(t, addonNotExist.IsProvisioningAllowed(namespace, collection))
	assert.True(t, addonManyProvision.IsProvisioningAllowed(namespace, collection))
}
