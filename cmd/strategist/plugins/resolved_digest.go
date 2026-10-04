package plugins

import (
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/application/digest"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

// ResolvedDigestOptions are the flags of `plugins resolved-digest`.
type ResolvedDigestOptions struct {
	Catalog  string
	Provider string
	Digest   string
	File     string
}

// NewResolvedDigest creates `plugins resolved-digest` (task 3.4 of
// .analysis/refined/20260927-embedded-weapon-channel-q01-q02/tasks.md): a
// read-only report of whether a delegated Ranger's self-reported
// weapon_invocation.resolved_digest matches the catalog's
// upstream_content_digest pin for that Weapon. It never invokes the Weapon
// and never reports a match when the pin is unavailable.
func NewResolvedDigest() *cobra.Command {
	opts := ResolvedDigestOptions{}
	cmd := &cobra.Command{
		Use:   "resolved-digest",
		Short: "Report whether a resolved Weapon file matches its catalog pin",
		Long: `Compares a Weapon's upstream_content_digest pin (internal/embed/defaults/
plugins/catalog.yaml) with a resolved digest — either a value already
computed by the caller (--resolved-digest, e.g. from a Ranger handoff's
weapon_invocation.resolved_digest) or a file this command hashes itself
(--file, sha256 of its raw bytes). It reads only; it never invokes the
Weapon and an unavailable pin is reported as pin_unavailable, never as a
match (see contracts/narrative/03-discovery.md § Weapon Profile for a
Delegated Ranger and design.md DEC-301/DEC-302 of the mission above).

Exits non-zero on a mismatch, so callers can gate on it directly. A missing
or unavailable pin, or a caller that supplied no digest, exits 0 and reports
the status: this command answers "what do we know", not "is this Weapon
trusted".`,
	}
	cmd.Flags().StringVar(&opts.Catalog, "catalog", "internal/embed/defaults/plugins/catalog.yaml", "path to the embedded plugin catalog")
	cmd.Flags().StringVar(&opts.Provider, "provider", "", "Weapon id to look up in the catalog (required)")
	cmd.Flags().StringVar(&opts.Digest, "resolved-digest", "", "an already-computed sha256:<64 hex> digest to compare (mutually exclusive with --file)")
	cmd.Flags().StringVar(&opts.File, "file", "", "path to a resolved Weapon file to hash and compare (mutually exclusive with --resolved-digest)")
	if err := cmd.MarkFlagRequired("provider"); err != nil {
		panic(err)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunResolvedDigest(cmd.OutOrStdout(), opts)
	}
	return cmd
}

// RunResolvedDigest resolves the digest to compare (from a flag or by
// hashing --file), compares it against the catalog pin, prints the result,
// and returns a non-nil error on mismatch so CI/scripts can gate on it.
func RunResolvedDigest(out io.Writer, opts ResolvedDigestOptions) error {
	resolvedDigest, err := digest.ResolveDigestInput(opts.Digest, opts.File, install.HashFileSHA256)
	if err != nil {
		return fmt.Errorf("resolved-digest: %w", err)
	}
	result, err := digest.CompareResolvedDigest(opts.Provider, resolvedDigest, func(provider, resolved string) (digest.ResolvedDigestComparison, error) {
		comparison, compareErr := install.CompareResolvedDigest(opts.Catalog, provider, resolved)
		if compareErr != nil {
			return digest.ResolvedDigestComparison{}, fmt.Errorf("compare resolved digest: %w", compareErr)
		}
		return digest.ResolvedDigestComparison{
			Status: digest.ResolvedDigestStatus(comparison.Status), Pin: comparison.Pin, Resolved: comparison.Resolved,
		}, nil
	})
	if err != nil {
		return fmt.Errorf("resolved-digest: %w", err)
	}
	if _, err := fmt.Fprintf(out, "resolved-digest: provider=%s status=%s pin=%s resolved=%s\n",
		opts.Provider, result.Status, displayOrNone(result.Pin), displayOrNone(result.Resolved)); err != nil {
		return fmt.Errorf("resolved-digest: write output: %w", err)
	}
	if result.Status == digest.ResolvedDigestMismatch {
		return fmt.Errorf("resolved-digest: mismatch for provider %q", opts.Provider)
	}
	return nil
}

func displayOrNone(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
