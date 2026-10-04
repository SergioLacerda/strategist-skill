package main

import (
	providerapp "github.com/SergioLacerda/strategist-skill/internal/application/provider"
	providerpkg "github.com/SergioLacerda/strategist-skill/internal/provider"
)

//nolint:dupl // bidirectional DTO adapters intentionally mirror the report fields.
func applicationProviderReport(report providerpkg.Report) providerapp.ProviderReport {
	reasons := mapProviderReasons(report.Reasons, func(reason providerpkg.Reason) providerapp.ProviderReason {
		return providerapp.ProviderReason{Code: reason.Code, Detail: reason.Detail}
	})
	return providerapp.ProviderReport{
		ProviderID: report.ProviderID, Version: report.Version, Source: report.Source,
		PackageDigest: report.PackageDigest, AdapterDigest: report.AdapterDigest,
		SupportedRoles: report.SupportedRoles, SupportedSlots: report.SupportedSlots,
		RequestedSlot: report.RequestedSlot, Readiness: report.Readiness,
		LiveInvocation: report.LiveInvocation, Reasons: reasons, Validated: report.Validated,
	}
}

func applicationProviderAddResult(result providerpkg.AddResult) providerapp.ProviderAddResult {
	return providerapp.ProviderAddResult{
		Report: applicationProviderReport(result.Report), InstanceID: result.InstanceID,
		BindingGeneration: result.BindingGeneration, TransactionState: result.TransactionState,
	}
}

//nolint:dupl // bidirectional DTO adapters intentionally mirror the report fields.
func providerReport(report providerapp.ProviderReport) providerpkg.Report {
	reasons := mapProviderReasons(report.Reasons, func(reason providerapp.ProviderReason) providerpkg.Reason {
		return providerpkg.Reason{Code: reason.Code, Detail: reason.Detail}
	})
	return providerpkg.Report{
		ProviderID: report.ProviderID, Version: report.Version, Source: report.Source,
		PackageDigest: report.PackageDigest, AdapterDigest: report.AdapterDigest,
		SupportedRoles: report.SupportedRoles, SupportedSlots: report.SupportedSlots,
		RequestedSlot: report.RequestedSlot, Readiness: report.Readiness,
		LiveInvocation: report.LiveInvocation, Reasons: reasons, Validated: report.Validated,
	}
}

func mapProviderReasons[S any, D any](reasons []S, convert func(S) D) []D {
	converted := make([]D, 0, len(reasons))
	for _, reason := range reasons {
		converted = append(converted, convert(reason))
	}
	return converted
}

func providerAddResult(result providerapp.ProviderAddResult) providerpkg.AddResult {
	return providerpkg.AddResult{
		Report: providerReport(result.Report), InstanceID: result.InstanceID,
		BindingGeneration: result.BindingGeneration, TransactionState: result.TransactionState,
	}
}
