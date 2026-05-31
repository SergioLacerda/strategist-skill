package i18n_test

import (
	"reflect"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func TestBundleForPTBR(t *testing.T) {
	b := i18n.BundleFor("pt-BR")
	assert.Equal(t, "(digitar outro...)", b.LabelCustomInput)
	assert.Equal(t, "Idioma da documentação", b.PromptDocLang)
}

func TestBundleForEN(t *testing.T) {
	b := i18n.BundleFor("en")
	assert.Equal(t, "(enter other...)", b.LabelCustomInput)
	assert.Equal(t, "Documentation language", b.PromptDocLang)
}

func TestBundleForUnknownDefaultsToEN(t *testing.T) {
	b := i18n.BundleFor("fr")
	assert.Equal(t, "(enter other...)", b.LabelCustomInput)
}

func TestBundleForCaseInsensitive(t *testing.T) {
	assert.Equal(t, i18n.BundleFor("pt-BR"), i18n.BundleFor("PT-BR"))
	assert.Equal(t, i18n.BundleFor("pt-BR"), i18n.BundleFor("pt-br"))
}

func TestAllFieldsNonEmptyEN(t *testing.T) {
	b := i18n.EN
	v := reflect.ValueOf(b)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i).Name
		assert.NotEmpty(t, v.Field(i).String(), "EN.%s must not be empty", field)
	}
}

func TestAllFieldsNonEmptyPT(t *testing.T) {
	b := i18n.PT
	v := reflect.ValueOf(b)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i).Name
		assert.NotEmpty(t, v.Field(i).String(), "PT.%s must not be empty", field)
	}
}
