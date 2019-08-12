// Code generated by mockery v1.0.0
package automock

import addons "github.com/kyma-project/helm-broker/internal/addons"

import mock "github.com/stretchr/testify/mock"

// AddonGetter is an autogenerated mock type for the AddonGetter type
type AddonGetter struct {
	mock.Mock
}

// Cleanup provides a mock function with given fields:
func (_m *AddonGetter) Cleanup() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCompleteAddon provides a mock function with given fields: entry
func (_m *AddonGetter) GetCompleteAddon(entry addons.EntryDTO) (addons.AddonDTO, error) {
	ret := _m.Called(entry)

	var r0 addons.AddonDTO
	if rf, ok := ret.Get(0).(func(addons.EntryDTO) addons.AddonDTO); ok {
		r0 = rf(entry)
	} else {
		r0 = ret.Get(0).(addons.AddonDTO)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(addons.EntryDTO) error); ok {
		r1 = rf(entry)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetIndex provides a mock function with given fields:
func (_m *AddonGetter) GetIndex() (*addons.IndexDTO, error) {
	ret := _m.Called()

	var r0 *addons.IndexDTO
	if rf, ok := ret.Get(0).(func() *addons.IndexDTO); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*addons.IndexDTO)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}