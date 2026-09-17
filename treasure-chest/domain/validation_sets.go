package domain

func stringSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func hasString(set map[string]struct{}, value string) bool {
	_, ok := set[value]
	return ok
}
