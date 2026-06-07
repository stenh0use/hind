package orchestration

import "regexp"

// versionTokenPattern matches the legacy package-version token format.
// Re-introduced after W-042 dropped pkg/build/release.ValidatePackageVersionToken
// when the import cycle was broken. Consumed by StartRequest.Validate() and by the
// CLI start command (pkg/cmd/hind/start) for pre-flight override validation.
var versionTokenPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

// IsValidVersionToken reports whether s matches the package-version token format.
// Callers must ensure s is non-empty; an empty string returns false.
// Exported so first-party callers (e.g., the CLI start command) can validate
// override input before constructing a StartRequest.
func IsValidVersionToken(s string) bool {
	return versionTokenPattern.MatchString(s)
}
