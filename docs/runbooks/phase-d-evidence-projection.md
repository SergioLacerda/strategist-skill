# Phase D evidence projection

Custom bindings remain owned by `.strategist/plugins.lock`. Ranked evidence is
owned by catalog, build, and probe inputs. The projection reports these
authorities; it never merges or repairs them.

Only live, explicit `certified` evidence can retain a certified projection.
Static-only evidence is projected as `unknown`; unavailable and failed states
remain non-certifying. A Custom/Ranked provider conflict is rejected without
fallback or lock mutation.
