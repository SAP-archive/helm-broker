package repository

import (
	"context"
	"fmt"
	"strings"

	"regexp"

	"net/url"

	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Template contains URL templating from secret logic
type Template struct {
	cli client.Client

	namespace string
}

var reg = regexp.MustCompile("\\{(.*?)\\}")

// NewTemplate returns a new Template service
func NewTemplate(cli client.Client) *Template {
	return &Template{
		cli: cli,
	}
}

// SetNamespace sets service's working namespace
func (t *Template) SetNamespace(namespace string) {
	t.namespace = namespace
}

// TemplateURL returns an URL to the repository with filled template fields
func (t *Template) TemplateURL(repository v1alpha1.SpecRepository) (string, error) {
	templateParameters := t.findURLTemplates(repository.URL)
	if len(templateParameters) == 0 {
		return repository.URL, nil
	}
	if repository.SecretRef == nil {
		return "", fmt.Errorf("template fields `%v` provided but secretRef is empty", templateParameters)
	}

	fetchNS := t.namespace
	if repository.SecretRef.Namespace != "" {
		fetchNS = repository.SecretRef.Namespace
	}

	secret := &v1.Secret{}
	err := t.cli.Get(context.Background(), types.NamespacedName{Namespace: fetchNS, Name: repository.SecretRef.Name}, secret)
	if err != nil {
		return "", errors.Wrapf(err, "while getting secret %s/%s", fetchNS, repository.SecretRef.Name)
	}

	result := repository.URL
	for _, val := range templateParameters {
		fieldName := val[1 : len(val)-1]
		tmp, ok := secret.Data[fieldName]
		if !ok {
			return "", fmt.Errorf("secret does not contain `%s` field", fieldName)
		}
		result = strings.Replace(result, val, url.QueryEscape(string(tmp)), -1)
	}

	return strings.Replace(result, "\n", "", -1), nil
}

func (t *Template) findURLTemplates(url string) []string {
	return reg.FindAllString(url, -1)
}
