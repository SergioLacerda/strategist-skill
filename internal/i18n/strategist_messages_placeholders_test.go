package i18n_test

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
)

var placeholderNameRE = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func TestRuntimeBundlesHaveMatchingPlaceholders(t *testing.T) {
	t.Parallel()

	en := reflect.ValueOf(i18n.ENRuntime)
	pt := reflect.ValueOf(i18n.PTBRRuntime)
	typ := en.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)
		enValue := en.Field(i).String()
		ptValue := pt.Field(i).String()

		assert.Empty(t, malformedPlaceholders(enValue), "ENRuntime.%s has malformed placeholder(s)", field.Name)
		assert.Empty(t, malformedPlaceholders(ptValue), "PTBRRuntime.%s has malformed placeholder(s)", field.Name)
		assert.Equal(t, placeholderSet(enValue), placeholderSet(ptValue),
			"Runtime placeholder parity mismatch for %s", field.Name)
	}
}

func placeholderSet(value string) []string {
	seen := map[string]struct{}{}
	for _, token := range extractPlaceholders(value) {
		seen[token] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for token := range seen {
		result = append(result, token)
	}
	sort.Strings(result)
	return result
}

func extractPlaceholders(value string) []string {
	var tokens []string
	for i := 0; i < len(value); i++ {
		if value[i] != '{' {
			continue
		}
		end := strings.IndexByte(value[i+1:], '}')
		if end < 0 {
			break
		}
		token := value[i+1 : i+1+end]
		if placeholderNameRE.MatchString(token) {
			tokens = append(tokens, token)
		}
		i += end + 1
	}
	return tokens
}

func malformedPlaceholders(value string) []string {
	var malformed []string
	for i := 0; i < len(value); {
		token, next, stop := nextMalformedPlaceholder(value, i)
		if token != "" {
			malformed = append(malformed, token)
		}
		if stop {
			return malformed
		}
		i = next
	}
	return malformed
}

func nextMalformedPlaceholder(value string, index int) (token string, next int, stop bool) {
	switch value[index] {
	case '}':
		return "}", index + 1, false
	case '{':
		return scanMalformedOpeningPlaceholder(value, index)
	default:
		return "", index + 1, false
	}
}

func scanMalformedOpeningPlaceholder(value string, index int) (token string, next int, stop bool) {
	end := strings.IndexByte(value[index+1:], '}')
	if end < 0 {
		return value[index:], len(value), true
	}
	name := value[index+1 : index+1+end]
	if placeholderNameRE.MatchString(name) {
		return "", index + end + 2, false
	}
	return "{" + name + "}", index + end + 2, false
}
