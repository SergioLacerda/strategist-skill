package install

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/wizard"
)

type nopCompiler struct{}

func (nopCompiler) CompileAll(_, _ string) error { return nil }

func newSvcW(t *testing.T, wizardInput string) Service {
	t.Helper()
	return Service{
		Extractor:      installtest.MinimalExtractor{},
		Compiler:       nopCompiler{},
		WizardPrompter: wizard.NewTextPrompter(strings.NewReader(wizardInput)),
		ShimHomeDir:    t.TempDir(),
	}
}

func p(input string) wizard.Prompter { return wizard.NewTextPrompter(strings.NewReader(input)) }
