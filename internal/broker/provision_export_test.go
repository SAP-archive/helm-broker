package broker

import (
	"testing"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func NewProvisionService(bg addonIDGetter, cg chartGetter, is instanceStorage, isg instanceStateGetter, oi operationInserter, ou operationUpdater,
	hi helmInstaller, oIDProv func() (internal.OperationID, error), log *logrus.Entry) *provisionService {
	return &provisionService{
		addonIDGetter:       bg,
		chartGetter:         cg,
		instanceGetter:      is,
		instanceInserter:    is,
		instanceStateGetter: isg,
		operationInserter:   oi,
		operationUpdater:    ou,
		operationIDProvider: oIDProv,
		helmInstaller:       hi,
		log:                 log,
	}
}

func (svc *provisionService) WithTestHookOnAsyncCalled(h func(internal.OperationID)) *provisionService {
	svc.testHookAsyncCalled = h
	return svc
}

func Test_createReleaseName(t *testing.T) {
	for name, tc := range map[string]struct {
		name     internal.AddonName
		planName internal.AddonPlanName
		ID       internal.InstanceID
		expected internal.ReleaseName
	}{
		"case #1": {
			name:     "test",
			planName: "test",
			ID:       "b1dc3be6-fcd2-4745-a473-5659554bb2b2",
			expected: "hb-test-test-b1dc3be6-fcd2-4745-a473-5659554bb2b2",
		},
		"case #2": {
			name:     "aaaaa-",
			planName: "bbbbb-",
			ID:       "b1dc3be6-fcd2-4745-a473-5659554bb2b2",
			expected: "hb-aaaaa-bbbbb-b1dc3be6-fcd2-4745-a473-5659554bb2b2",
		},
		"case #3": {
			name:     "name-longer-than-six-chars",
			planName: "test",
			ID:       "321d0f58-8632-4fe9-92f6-fc5694695cf5",
			expected: "hb-name-l-test-321d0f58-8632-4fe9-92f6-fc5694695cf5",
		},
		"case #4": {
			name:     "test",
			planName: "plan-name-longer-than-six-chars",
			ID:       "8d2cb4da-fda1-4e32-86e1-f1b6f7d79dea",
			expected: "hb-test-plan-n-8d2cb4da-fda1-4e32-86e1-f1b6f7d79dea",
		},
		"case #5": {
			name:     "name-longer-than-six-chars",
			planName: "plan-name-longer-than-six-chars",
			ID:       "d941f4d0-7ae7-437e-b1a8-41e5b1ded71d",
			expected: "hb-name-l-plan-n-d941f4d0-7ae7-437e-b1a8-41e5b1ded71d",
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := createReleaseName(tc.name, tc.planName, tc.ID)

			assert.True(t, len(result) <= 53)
			assert.Equal(t, tc.expected, result)
		})
	}
}
