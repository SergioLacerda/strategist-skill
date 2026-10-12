package installtest

import (
	"fmt"
)

// AnswerPrompter answers Select prompts from a fixed script and records what it
// was asked, so a test can assert both the question and its default.
type AnswerPrompter struct {
	Answer   string
	Titles   []string
	Defaults []string
	Options  [][]string
}

// Select answers from the script and records the question.
func (a *AnswerPrompter) Select(title, defaultVal string, options []string) (string, error) {
	a.Titles, a.Defaults, a.Options = append(a.Titles, title), append(a.Defaults, defaultVal), append(a.Options, options)
	if a.Answer != "" {
		return a.Answer, nil
	}
	return defaultVal, nil
}

// Input accepts the default.
func (a *AnswerPrompter) Input(_, defaultVal string) (string, error) { return defaultVal, nil }

// SelectOrInput accepts the default.
func (a *AnswerPrompter) SelectOrInput(_, defaultVal string, _ []string, _ string) (string, error) {
	return defaultVal, nil
}

// DocumentationRootPrompter answers Input prompts from a scripted list of values and accepts every default.
type DocumentationRootPrompter struct {
	Values []string
}

// Select accepts the default.
func (p *DocumentationRootPrompter) Select(_, defaultVal string, _ []string) (string, error) {
	return defaultVal, nil
}

// Input returns the next scripted value.
func (p *DocumentationRootPrompter) Input(_, _ string) (string, error) {
	if len(p.Values) == 0 {
		return "", fmt.Errorf("DocumentationRootPrompter: no scripted value")
	}
	value := p.Values[0]
	p.Values = p.Values[1:]
	return value, nil
}

// SelectOrInput accepts the default.
func (p *DocumentationRootPrompter) SelectOrInput(_, defaultVal string, _ []string, _ string) (string, error) {
	return defaultVal, nil
}
