package mission

import "time"

func builtinResolverRegistry(strategistRoot, basePath string, now time.Time) (*ResolverRegistry, error) {
	return NewResolverRegistry(
		documentationExecutionResolver{},
		adrExecutionResolver{strategistRoot: strategistRoot, basePath: basePath, now: fixedTimeProvider{value: now}},
		runbookExecutionResolver{strategistRoot: strategistRoot, now: fixedTimeProvider{value: now}},
	)
}
