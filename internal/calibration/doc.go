// Package calibration is the pure evaluation core of the DOJO judge calibration
// cases (ADR-0066). It generates labeled cases from packages (mutations whose
// label is known by construction), runs them against a judge behind a port,
// records raw observations, and computes rates with exact confidence intervals,
// fitness gates and a risk-controlled threshold recommendation. It performs no
// file or network I/O: the DOJO supplies the corpus and persists the results.
package calibration
