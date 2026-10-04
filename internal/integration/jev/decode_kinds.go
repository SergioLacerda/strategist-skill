package jev

import "github.com/SergioLacerda/strategist-skill/internal/integration"

func inUnit(value float64) bool { return value >= 0 && value <= 1 }

func convertNoul(raw wireAnswer) (integration.Answer, error) {
	if raw.Noul == nil || !inUnit(*raw.Noul) || raw.Confidence != nil || raw.Probabilities != nil || raw.Choice != nil || raw.Score != nil {
		return integration.Answer{}, invalid("noul answer must carry only a value in [0,1]")
	}
	return integration.Answer{Kind: integration.KindNoul, Noul: *raw.Noul}, nil
}

func convertChoice(q integration.Question, raw wireAnswer) (integration.Answer, error) {
	if raw.Choice == nil || raw.Confidence == nil || !inUnit(*raw.Confidence) || raw.Noul != nil || raw.Score != nil {
		return integration.Answer{}, invalid("choice answer needs a choice and a confidence in [0,1]")
	}
	offered := offeredKeys(q)
	if !offered[*raw.Choice] || !probabilitiesWithin(raw.Probabilities, offered) {
		return integration.Answer{}, invalid("choice or probabilities are outside the offered options")
	}
	return integration.Answer{Kind: integration.KindChoice, Choice: *raw.Choice, Confidence: *raw.Confidence, HasConfidence: true}, nil
}

func offeredKeys(q integration.Question) map[string]bool {
	offered := make(map[string]bool, len(q.Options))
	for _, option := range q.Options {
		offered[option.Key] = true
	}
	return offered
}

func probabilitiesWithin(probabilities map[string]float64, offered map[string]bool) bool {
	for key, probability := range probabilities {
		if !offered[key] || !inUnit(probability) {
			return false
		}
	}
	return true
}

func convertScore(q integration.Question, raw wireAnswer) (integration.Answer, error) {
	if raw.Score == nil || raw.Confidence == nil || !inUnit(*raw.Confidence) || raw.Noul != nil || raw.Choice != nil {
		return integration.Answer{}, invalid("score answer needs a score and a confidence in [0,1]")
	}
	if *raw.Score < 0 || *raw.Score > float64(len(q.Labels)-1) {
		return integration.Answer{}, invalid("score is outside the offered labels")
	}
	return integration.Answer{Kind: integration.KindScore, Score: *raw.Score, Confidence: *raw.Confidence, HasConfidence: true}, nil
}
