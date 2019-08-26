package repository

import (
	"fmt"
	"testing"

	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	secretRef = "test"

	keyIDField     = "KEY_ID"
	keySecretField = "KEY_SECRET"

	testData = "fix"
)

func TestTemplate_HappyPath(t *testing.T) {
	for tn, tc := range map[string]struct {
		specRepository v1alpha1.SpecRepository
		objs           []runtime.Object
		expURL         string
	}{
		"only-field": {
			specRepository: v1alpha1.SpecRepository{
				URL: fmt.Sprintf("{%s}", keyIDField),
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			objs:   []runtime.Object{fixSecret()},
			expURL: fmt.Sprintf("%s", testData),
		},
		"one-field": {
			specRepository: v1alpha1.SpecRepository{
				URL: fmt.Sprintf("s3://index/path?access_key_id={%s}", keyIDField),
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			objs:   []runtime.Object{fixSecret()},
			expURL: fmt.Sprintf("s3://index/path?access_key_id=%s", testData),
		},
		"many-fields": {
			specRepository: v1alpha1.SpecRepository{
				URL: fmt.Sprintf("s3://index/path?access_key_id={%s}&access_key_secret={%s}", keyIDField, keySecretField),
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			objs:   []runtime.Object{fixSecret()},
			expURL: fmt.Sprintf("s3://index/path?access_key_id=%s&access_key_secret=%s", testData, testData),
		},
		"no-fields": {
			specRepository: v1alpha1.SpecRepository{
				URL: "s3://index/path?access_key_id=fix",
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			objs:   []runtime.Object{fixSecret()},
			expURL: "s3://index/path?access_key_id=fix",
		},
		"no-fields-no-secret": {
			specRepository: v1alpha1.SpecRepository{
				URL: "s3://index/path?access_key_id=fix",
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			expURL: "s3://index/path?access_key_id=fix",
		},
		"mixed-fields": {
			specRepository: v1alpha1.SpecRepository{
				URL: fmt.Sprintf("s3://index/path?access_key_id={%s}&ref=master&access_key_secret={%s}", keyIDField, keySecretField),
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			objs:   []runtime.Object{fixSecret()},
			expURL: fmt.Sprintf("s3://index/path?access_key_id=%s&ref=master&access_key_secret=%s", testData, testData),
		},
	} {
		t.Run(tn, func(t *testing.T) {
			c := fake.NewFakeClient(tc.objs...)
			tmp := NewTemplate(c)

			result, err := tmp.TemplateURL(tc.specRepository)
			require.NoError(t, err)
			assert.Equal(t, tc.expURL, result)
		})
	}

}

func TestTemplate_FillNamespaceIfNotProvided(t *testing.T) {
	c := fake.NewFakeClient(fixSecret())
	tmp := NewTemplate(c)

	tmp.SetNamespace(secretRef)
	result, err := tmp.TemplateURL(v1alpha1.SpecRepository{
		URL: fmt.Sprintf("{%s}", keyIDField),
		SecretRef: &v1.SecretReference{
			Name: secretRef,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, testData, result)
}

func TestTemplate_Error(t *testing.T) {
	for tn, tc := range map[string]struct {
		specRepository v1alpha1.SpecRepository
		objs           []runtime.Object
		errMsg         string
	}{
		"missing-secret": {
			specRepository: v1alpha1.SpecRepository{
				URL: "access_key_id={KEY_NAME_FROM_SECRET_PLACEHOLDER}",
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			errMsg: "while getting secret test/test: secrets \"test\" not found",
		},
		"missing-field": {
			specRepository: v1alpha1.SpecRepository{
				URL: "access_key_id={KEY_NAME_FROM_SECRET_PLACEHOLDER}",
				SecretRef: &v1.SecretReference{
					Name:      secretRef,
					Namespace: secretRef,
				},
			},
			objs: []runtime.Object{
				fixSecret(),
			},
			errMsg: "secret does not contain `KEY_NAME_FROM_SECRET_PLACEHOLDER` field",
		},
	} {
		t.Run(tn, func(t *testing.T) {
			c := fake.NewFakeClient(tc.objs...)
			tmp := NewTemplate(c)

			_, err := tmp.TemplateURL(tc.specRepository)
			require.EqualError(t, err, tc.errMsg)
		})
	}

}

func fixSecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretRef,
			Namespace: secretRef,
		},
		Data: map[string][]byte{
			keyIDField:     []byte(testData),
			keySecretField: []byte(testData),
		},
	}
}
