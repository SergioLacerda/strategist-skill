package handoff

// policyInvalidResult builds the Result for a policy that failed
// ValidatePolicy (F-H4, ADR-0057 § design.md task 2.5). It never sets
// NextAction: policy.OnFailure belongs to the policy that was just declared
// invalid, and reusing it here would route a configuration error to
// Archivist as if it were a repairable semantic failure. Split out of
// verify.go to keep that file under the repo's file-size budget.
func policyInvalidResult(err error) Result {
	return Result{
		Status:       StatusPolicyInvalid,
		Passed:       false,
		PolicyErrors: policyErrorMessages(err),
	}
}

// policyErrorMessages splits a joined ValidatePolicy error (errors.Join)
// back into its individual messages when possible, so PolicyErrors lists one
// distinct problem per entry rather than one multi-line string. Falls back
// to the error's own message when it does not expose the multi-error
// Unwrap() []error interface.
func policyErrorMessages(err error) []string {
	type unwrapper interface{ Unwrap() []error }
	joined, ok := err.(unwrapper)
	if !ok {
		return []string{err.Error()}
	}
	errs := joined.Unwrap()
	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		messages = append(messages, e.Error())
	}
	return messages
}
