package broker_test

import (
	"fmt"
	"testing"

	"github.com/kyma-project/helm-broker/internal/storage/driver/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/broker"
	"github.com/kyma-project/helm-broker/internal/broker/automock"
	"github.com/pkg/errors"
)

func newInstanceStateServiceTestSuite(t *testing.T) *instanceStateServiceTestSuite {
	return &instanceStateServiceTestSuite{t: t}
}

type instanceStateServiceTestSuite struct {
	t   *testing.T
	Exp expAll
}

func (ts *instanceStateServiceTestSuite) SetUp() {
	ts.Exp.Populate()
}

func TestInstanceStateServiceIsProvisioned(t *testing.T) {
	for sym, tc := range map[string]struct {
		genOps func(ts *instanceStateServiceTestSuite) []*internal.InstanceOperation
		exp    bool
	}{
		"true/singleCreateSucceeded": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				return append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
			},
			exp: true,
		},
		"true/CreateSucceededThanRemoveInProgress": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateInProgress))
				return out
			},
			exp: true,
		},
		"false/singleCreateInProgress": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				return append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateInProgress))
			},
			exp: false,
		},
		"false/CreateSucceededThanRemoveSucceeded": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateSucceeded))
				return out
			},
			exp: false,
		},
	} {
		t.Run(fmt.Sprintf("Success/%s", sym), func(t *testing.T) {
			// GIVEN
			ts := newInstanceStateServiceTestSuite(t)
			ts.SetUp()

			ocgMock := &automock.OperationStorage{}
			defer ocgMock.AssertExpectations(t)
			ocgMock.On("GetAll", ts.Exp.InstanceID).Return(tc.genOps(ts), nil).Once()

			svc := broker.NewInstanceStateService(ocgMock)

			// WHEN
			got, err := svc.IsProvisioned(ts.Exp.InstanceID)

			// THEN
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, got)
		})
	}

	t.Run("Success/false/InstanceNotFound", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, notFoundError{}).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		got, err := svc.IsProvisioned(ts.Exp.InstanceID)

		// THEN
		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("Failure/GenericStorageError", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		fixErr := errors.New("fix-storage-error")
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, fixErr).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		got, err := svc.IsProvisioned(ts.Exp.InstanceID)

		// THEN
		assert.EqualError(t, err, fmt.Sprintf("while getting instance operations from storage: %s", fixErr.Error()))
		assert.False(t, got)
	})
}

func TestInstanceStateServiceIsDeprovisioned(t *testing.T) {
	for sym, tc := range map[string]struct {
		genOps func(ts *instanceStateServiceTestSuite) []*internal.InstanceOperation
		exp    bool
	}{
		"true/singleRemoveSucceeded": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				return append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateSucceeded))
			},
			exp: true,
		},
		"true/CreateSucceededThanRemoveInProgress": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateInProgress))
				return out
			},
			exp: false,
		},
		"false/singleRemoveInProgress": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				return append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateInProgress))
			},
			exp: false,
		},
		"false/CreateSucceededThanRemoveSucceeded": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateSucceeded))
				return out
			},
			exp: true,
		},
	} {
		t.Run(fmt.Sprintf("Success/%s", sym), func(t *testing.T) {
			// GIVEN
			ts := newInstanceStateServiceTestSuite(t)
			ts.SetUp()

			ocgMock := &automock.OperationStorage{}
			defer ocgMock.AssertExpectations(t)
			ocgMock.On("GetAll", ts.Exp.InstanceID).Return(tc.genOps(ts), nil).Once()

			svc := broker.NewInstanceStateService(ocgMock)

			// WHEN
			got, err := svc.IsDeprovisioned(ts.Exp.InstanceID)

			// THEN
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, got)
		})
	}

	t.Run("Success/false/InstanceNotFound", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, notFoundError{}).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		got, err := svc.IsDeprovisioned(ts.Exp.InstanceID)

		// THEN
		assert.True(t, broker.IsNotFoundError(err))
		assert.False(t, got)
	})

	t.Run("Failure/GenericStorageError", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		fixErr := errors.New("fix-storage-error")
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, fixErr).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		got, err := svc.IsDeprovisioned(ts.Exp.InstanceID)

		// THEN
		assert.EqualError(t, err, fmt.Sprintf("while getting instance operations from storage: %s", fixErr.Error()))
		assert.False(t, got)
	})
}

func TestInstanceStateServiceIsDeprovisioningInProgress(t *testing.T) {
	for sym, tc := range map[string]struct {
		genOps func(ts *instanceStateServiceTestSuite) []*internal.InstanceOperation
		exp    bool
	}{
		"false/singleRemoveSucceeded": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				return append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateSucceeded))
			},
			exp: false,
		},
		"true/CreateSucceededThanRemoveInProgress": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateInProgress))
				return out
			},
			exp: true,
		},
		"false/NoOp": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) { return out },
			exp:    false,
		},
		"false/CreateSucceededThanRemoveSucceeded": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateSucceeded))
				return out
			},
			exp: false,
		},
	} {
		t.Run(fmt.Sprintf("Success/%s", sym), func(t *testing.T) {
			// GIVEN
			ts := newInstanceStateServiceTestSuite(t)
			ts.SetUp()

			ocgMock := &automock.OperationStorage{}
			defer ocgMock.AssertExpectations(t)
			ocgMock.On("GetAll", ts.Exp.InstanceID).Return(tc.genOps(ts), nil).Once()

			svc := broker.NewInstanceStateService(ocgMock)

			// WHEN
			gotOpID, gotInProgress, err := svc.IsDeprovisioningInProgress(ts.Exp.InstanceID)

			// THEN
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, gotInProgress)
			if tc.exp {
				assert.Equal(t, ts.Exp.OperationID, gotOpID)
			}
		})
	}

	t.Run("Success/false/InstanceNotFound", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, notFoundError{}).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		gotOpID, got, err := svc.IsDeprovisioningInProgress(ts.Exp.InstanceID)

		// THEN
		assert.NoError(t, err)
		assert.False(t, got)
		assert.Zero(t, gotOpID)
	})

	t.Run("Failure/GenericStorageError", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		fixErr := errors.New("fix-storage-error")
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, fixErr).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		gotOpID, got, err := svc.IsDeprovisioningInProgress(ts.Exp.InstanceID)

		// THEN
		assert.EqualError(t, err, fmt.Sprintf("while getting instance operations from storage: %s", fixErr.Error()))
		assert.False(t, got)
		assert.Zero(t, gotOpID)
	})
}

func TestInstanceStateServiceIsProvisioningInProgress(t *testing.T) {
	for sym, tc := range map[string]struct {
		genOps        func(ts *instanceStateServiceTestSuite) []*internal.InstanceOperation
		expInProgress bool
	}{
		"true/singleCreateInProgress": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				return append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateInProgress))
			},
			expInProgress: true,
		},
		"false/singleCreateSucceeded": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				return append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
			},
			expInProgress: false,
		},
		"false/NoOp": {
			genOps:        func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) { return out },
			expInProgress: false,
		},
		"false/CreateSucceededThanRemoveInProgress": {
			genOps: func(ts *instanceStateServiceTestSuite) (out []*internal.InstanceOperation) {
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewInstanceOperation(internal.OperationTypeRemove, internal.OperationStateInProgress))
				return out
			},
			expInProgress: false,
		},
	} {
		t.Run(fmt.Sprintf("Success/%s", sym), func(t *testing.T) {
			// GIVEN
			ts := newInstanceStateServiceTestSuite(t)
			ts.SetUp()

			ocgMock := &automock.OperationStorage{}
			defer ocgMock.AssertExpectations(t)
			ocgMock.On("GetAll", ts.Exp.InstanceID).Return(tc.genOps(ts), nil).Once()

			svc := broker.NewInstanceStateService(ocgMock)

			// WHEN
			gotOpID, gotInProgress, err := svc.IsProvisioningInProgress(ts.Exp.InstanceID)

			// THEN
			assert.NoError(t, err)
			assert.Equal(t, tc.expInProgress, gotInProgress)
			if tc.expInProgress {
				assert.Equal(t, ts.Exp.OperationID, gotOpID)
			}
		})
	}

	t.Run("Success/false/InstanceNotFound", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, notFoundError{}).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		gotOpID, got, err := svc.IsProvisioningInProgress(ts.Exp.InstanceID)

		// THEN
		assert.NoError(t, err)
		assert.False(t, got)
		assert.Zero(t, gotOpID)
	})

	t.Run("Failure/GenericStorageError", func(t *testing.T) {
		// GIVEN
		ts := newInstanceStateServiceTestSuite(t)
		ts.SetUp()

		ocgMock := &automock.OperationStorage{}
		defer ocgMock.AssertExpectations(t)
		fixErr := errors.New("fix-storage-error")
		ocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, fixErr).Once()

		svc := broker.NewInstanceStateService(ocgMock)

		// WHEN
		gotOpID, got, err := svc.IsProvisioningInProgress(ts.Exp.InstanceID)

		// THEN
		assert.EqualError(t, err, fmt.Sprintf("while getting instance operations from storage: %s", fixErr.Error()))
		assert.False(t, got)
		assert.Zero(t, gotOpID)
	})
}

func newBindStateServiceTestSuite(t *testing.T) *bindStateServiceTestSuite {
	return &bindStateServiceTestSuite{t: t}
}

type bindStateServiceTestSuite struct {
	t   *testing.T
	Exp expAll
}

func (ts *bindStateServiceTestSuite) SetUp() {
	ts.Exp.Populate()
}

func TestBindStateServiceIsBound(t *testing.T) {
	for sym, tc := range map[string]struct {
		genOps func(ts *bindStateServiceTestSuite) []*internal.BindOperation
		exp    bool
	}{
		"true/singleCreateSucceeded": {
			genOps: func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) {
				return append(out, ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
			},
			exp: true,
		},
		"true/CreateSucceededThanRemoveInProgress": {
			genOps: func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) {
				out = append(out, ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewBindOperation(internal.OperationTypeRemove, internal.OperationStateInProgress))
				return out
			},
			exp: true,
		},
		"false/singleCreateInProgress": {
			genOps: func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) {
				return append(out, ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateInProgress))
			},
			exp: false,
		},
		"false/CreateSucceededThanRemoveSucceeded": {
			genOps: func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) {
				out = append(out, ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewBindOperation(internal.OperationTypeRemove, internal.OperationStateSucceeded))
				return out
			},
			exp: false,
		},
	} {
		t.Run(fmt.Sprintf("Success/%s", sym), func(t *testing.T) {
			// GIVEN
			ts := newBindStateServiceTestSuite(t)
			ts.SetUp()

			bocgMock := &automock.BindOperationStorage{}
			defer bocgMock.AssertExpectations(t)
			bocgMock.On("GetAll", ts.Exp.InstanceID).Return(tc.genOps(ts), nil).Once()

			svc := broker.NewBindStateService(bocgMock)

			// WHEN
			_, got, err := svc.IsBound(ts.Exp.InstanceID, ts.Exp.BindingID)

			// THEN
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, got)
		})
	}

	t.Run("Success/false/InstanceNotFound", func(t *testing.T) {
		// GIVEN
		ts := newBindStateServiceTestSuite(t)
		ts.SetUp()

		svc := broker.NewBindStateService(memory.NewBindOperation())

		// WHEN
		_, got, err := svc.IsBound(ts.Exp.InstanceID, ts.Exp.BindingID)

		// THEN
		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("Failure/GenericStorageError", func(t *testing.T) {
		// GIVEN
		ts := newBindStateServiceTestSuite(t)
		ts.SetUp()

		bocgMock := &automock.BindOperationStorage{}
		defer bocgMock.AssertExpectations(t)
		fixErr := errors.New("fix-storage-error")
		bocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, fixErr).Once()

		svc := broker.NewBindStateService(bocgMock)

		// WHEN
		_, got, err := svc.IsBound(ts.Exp.InstanceID, ts.Exp.BindingID)

		// THEN
		assert.EqualError(t, err, fmt.Sprintf("while getting bind operations from storage: %s", fixErr.Error()))
		assert.False(t, got)
	})
}

func TestBindStateServiceIsBindingInProgress(t *testing.T) {
	for sym, tc := range map[string]struct {
		genOps        func(ts *bindStateServiceTestSuite) []*internal.BindOperation
		expInProgress bool
	}{
		"true/singleCreateInProgress": {
			genOps: func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) {
				return append(out, ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateInProgress))
			},
			expInProgress: true,
		},
		"false/singleCreateSucceeded": {
			genOps: func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) {
				return append(out, ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
			},
			expInProgress: false,
		},
		"false/NoOp": {
			genOps:        func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) { return out },
			expInProgress: false,
		},
		"false/CreateSucceededThanRemoveInProgress": {
			genOps: func(ts *bindStateServiceTestSuite) (out []*internal.BindOperation) {
				out = append(out, ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded))
				out = append(out, ts.Exp.NewBindOperation(internal.OperationTypeRemove, internal.OperationStateInProgress))
				return out
			},
			expInProgress: false,
		},
	} {
		t.Run(fmt.Sprintf("Success/%s", sym), func(t *testing.T) {
			// GIVEN
			ts := newBindStateServiceTestSuite(t)
			ts.SetUp()

			bocgMock := &automock.BindOperationStorage{}
			defer bocgMock.AssertExpectations(t)
			bocgMock.On("GetAll", ts.Exp.InstanceID).Return(tc.genOps(ts), nil).Once()

			svc := broker.NewBindStateService(bocgMock)

			// WHEN
			gotOpID, gotInProgress, err := svc.IsBindingInProgress(ts.Exp.InstanceID, ts.Exp.BindingID)

			// THEN
			assert.NoError(t, err)
			assert.Equal(t, tc.expInProgress, gotInProgress)
			if tc.expInProgress {
				assert.Equal(t, ts.Exp.OperationID, gotOpID)
			}
		})
	}

	t.Run("Success/false/InstanceNotFound", func(t *testing.T) {
		// GIVEN
		ts := newBindStateServiceTestSuite(t)
		ts.SetUp()

		svc := broker.NewBindStateService(memory.NewBindOperation())

		// WHEN
		gotOpID, got, err := svc.IsBindingInProgress(ts.Exp.InstanceID, ts.Exp.BindingID)

		// THEN
		assert.NoError(t, err)
		assert.False(t, got)
		assert.Zero(t, gotOpID)
	})

	t.Run("Failure/GenericStorageError", func(t *testing.T) {
		// GIVEN
		ts := newBindStateServiceTestSuite(t)
		ts.SetUp()

		bocgMock := &automock.BindOperationStorage{}
		defer bocgMock.AssertExpectations(t)
		fixErr := errors.New("fix-storage-error")
		bocgMock.On("GetAll", ts.Exp.InstanceID).Return(nil, fixErr).Once()

		svc := broker.NewBindStateService(bocgMock)

		// WHEN
		gotOpID, got, err := svc.IsBindingInProgress(ts.Exp.InstanceID, ts.Exp.BindingID)

		// THEN
		assert.EqualError(t, err, fmt.Sprintf("while getting bind operations from storage: %s", fixErr.Error()))
		assert.False(t, got)
		assert.Zero(t, gotOpID)
	})
}
