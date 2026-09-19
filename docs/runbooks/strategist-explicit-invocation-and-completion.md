# Strategist Explicit Invocation and Mission Completion

## Activation Boundary

Strategist is active for a request only when the requester explicitly invokes the
Strategist skill, a host-provided Strategist slash command, or a Strategist CLI
mission operation. The existence of `.strategist/` in a workspace does not activate
the skill.

`strategist check --json` is a preflight operation. It reports runtime and binding
readiness; it does not start a Strategist mission, invoke a provider, or prove that a
provider was invoked by the host.

## During a Mission

After an explicit invocation, follow the declared mission pipeline. An accepted
approval gate authorizes only the documentation targets listed in the accepted
refined package. It does not authorize code, tests, configuration, hooks, or Git
changes that appear as implementation handoffs.

## Terminal Completion

When Sniper has materialized every approved documentation target, the mission status
is `documentation_applied`. The terminal response must name the completed Strategist
mission and state that Strategist has finished for that mission.

This is documentation-materialization completion, not evidence that related code
handoffs are implemented or validated.

## Follow-up Work

A later request is independent of the completed mission. It is evaluated according to
its own explicit request and authorization; the requester does not need to ask to
leave or disable Strategist. No approval from the completed mission carries over to
another skill, CLI command, or implementation task.
