package eval

func paramString(params map[string]any, key string) string {
	v, ok := params[key].(string)
	if !ok {
		return ""
	}
	return v
}

func paramBool(params map[string]any, key string) bool {
	v, ok := params[key].(bool)
	if !ok {
		return false
	}
	return v
}

func paramStringSlice(params map[string]any, key string) []string {
	raw, ok := params[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if str, ok := v.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
