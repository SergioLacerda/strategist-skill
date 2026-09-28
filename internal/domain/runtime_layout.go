package domain

// RuntimeLayoutGeneration identifies the layout of the installed runtime tree, as
// this binary writes it. It is a monotonic integer, not a release version: builds
// tagged dev or -dirty have no comparable semver, so the layout is versioned on its
// own.
//
//	0  legacy or absent: a manifest written before this marker existed
//	1  generation N-1: layout-aware; the generated compat view is still written
//	2  generation N: the compat view is no longer written or shipped
//
// The commit that changes the layout also bumps this constant, and a test pins its
// value. It is copied into the install manifest (InstallManifest.RuntimeLayoutGeneration)
// and never enters a digest input: it is not part of the generated skill manifest,
// the catalog stamps, the lock files or the ranked certification pins.
const RuntimeLayoutGeneration = 1
