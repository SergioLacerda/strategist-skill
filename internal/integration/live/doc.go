// Package live holds the opt-in live check of the provider integration. Its
// tests carry the "live" build tag, are never part of the default suite and call
// the real provider with the credential resolved from the workspace .env, only
// ever with synthetic text that contains nothing of a mission.
//
// They also skip unless STRATEGIST_LIVE_CONSENT=yes. The sanctioned path is the DOJO:
// `strategist dojo calibrate jev-live` (consent term, then --consent). Non-test code
// here (Smoke) is what that DOJO case runs; it is hermetic when its judge is simulated.
package live
