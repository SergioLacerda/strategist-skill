package trust

func missingReasons(code string, required, present []string) []Reason {
	var reasons []Reason
	for _, item := range required {
		if !contains(present, item) {
			reasons = append(reasons, Reason{Code: code, Detail: item})
		}
	}
	return reasons
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
