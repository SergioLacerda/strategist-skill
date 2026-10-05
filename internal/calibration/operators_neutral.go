package calibration

// neutralOperators change layout, never meaning. They must stay positive, so they
// measure false rejection and the judge's sensitivity to noise.
func neutralOperators() []Operator {
	neutral := func(name string, apply func(string) string) Operator {
		return Operator{
			Name: name, Stage: 1, Boundary: BoundaryAny, Band: BandClean, Expect: ExpectAccept, Neutral: true,
			Applies: func(Fields) bool { return true },
			Apply:   func(f Fields, _ Context) Fields { return mapFields(f, apply) },
		}
	}
	return []Operator{
		neutral("collapse_spaces", func(s string) string { return manySpaces.ReplaceAllString(s, " ") }),
		neutral("collapse_blank_lines", func(s string) string { return manyBreaks.ReplaceAllString(s, "\n\n") }),
		neutral("trim_trailing_space", func(s string) string { return trailingWS.ReplaceAllString(s, "") }),
	}
}

// BoundaryAny marks an operator valid for every boundary.
const BoundaryAny = "any"

func mapFields(fields Fields, edit func(string) string) Fields {
	out := make(Fields, len(fields))
	for name, value := range fields {
		out[name] = edit(value)
	}
	return out
}
