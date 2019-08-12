package addons_test

import (
	"errors"
	"testing"

	"github.com/kyma-project/helm-broker/internal/addons"
	"github.com/stretchr/testify/assert"
)

func TestLoadingError(t *testing.T) {
	tests := map[string]struct {
		givenErr          error
		expToBeLoadingErr bool
	}{
		"Should report true for Loading error": {
			givenErr:          addons.NewLoadingError(errors.New("fix err")),
			expToBeLoadingErr: true,
		},
		"Should report false for generic error": {
			givenErr:          errors.New("fix err"),
			expToBeLoadingErr: false,
		},
		"Should report false for Fetching error": {
			givenErr:          addons.NewFetchingError(errors.New("fix err")),
			expToBeLoadingErr: false,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			assert.Equal(t, tc.expToBeLoadingErr, addons.IsLoadingError(tc.givenErr))
		})
	}
}

func TestFetchingError(t *testing.T) {
	tests := map[string]struct {
		givenErr          error
		expToBeLoadingErr bool
	}{
		"Should report true for Fetching error": {
			givenErr:          addons.NewFetchingError(errors.New("fix err")),
			expToBeLoadingErr: true,
		},
		"Should report false for generic error": {
			givenErr:          errors.New("fix err"),
			expToBeLoadingErr: false,
		},
		"Should report false for Loading error": {
			givenErr:          addons.NewLoadingError(errors.New("fix err")),
			expToBeLoadingErr: false,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			assert.Equal(t, tc.expToBeLoadingErr, addons.IsFetchingError(tc.givenErr))
		})
	}
}
