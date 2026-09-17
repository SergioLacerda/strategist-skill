# Runbook: Truthful Runtime Probing

## Boundary

Package resolution and static manifest validation prove that an artifact is
structurally usable. They do not prove that the host can invoke the skill.
`LocalPathConnector` therefore reports `probe_not_verified` with a non-ready
status for valid static inputs. Only a host-specific connector that owns and
observes a live health or invocation check may report `ready`.

## Migration rule

Role/provider migration must remain fail-closed when the connector returns
`unknown`, `unsupported`, or `blocked`. The previous last-known-good binding
must remain active, and the connector reason must remain available in the
lifecycle journal for diagnosis.

Do not add a fake invocation, network call, or host-loader dependency to make a
static connector appear ready. When live host evidence becomes available,
implement it in a connector-owned boundary and retain separate structural,
catalog, and runtime readiness dimensions.
