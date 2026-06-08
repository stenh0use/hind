// Package overrides implements three-level version source precedence
// (CLI flag > env var > compiled-in default) for hind CLI commands that accept
// per-package version overrides for the three HashiCorp services (Nomad,
// Consul, Vault).
//
// Resolution runs at the CLI boundary, before any Docker or container side
// effects, so invalid input fails fast and error messages can attribute to the
// exact offending source (flag name vs. environment variable name).
package overrides

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/stenh0use/hind/pkg/cluster/orchestration"
)

// Source identifies which input layer supplied an effective version.
type Source int

const (
	SourceDefault Source = iota
	SourceEnv
	SourceFlag
)

// Env var names. Exported so tests and production code agree.
const (
	EnvNomad  = "NOMAD_VERSION"
	EnvConsul = "CONSUL_VERSION"
	EnvVault  = "VAULT_VERSION"
)

// Flag names. Exported only for use in error message attribution.
const (
	FlagNomad  = "--nomad-version"
	FlagConsul = "--consul-version"
	FlagVault  = "--vault-version"
)

// Resolved is the per-package result of three-level resolution.
// Value is empty when Source == SourceDefault (caller falls back to release defaults).
type Resolved struct {
	Value  string
	Source Source
}

// Set bundles the three package outputs.
type Set struct {
	Nomad  Resolved
	Consul Resolved
	Vault  Resolved
}

// FlagInputs is the CLI-parsed input passed to Resolve.
// *Set fields mirror cobra.Flags().Changed(name) and disambiguate "flag unset"
// from "flag set to empty string". An explicit `--nomad-version ""` must fail
// validation, not silently fall back to env.
type FlagInputs struct {
	NomadVersion  string
	ConsulVersion string
	VaultVersion  string
	NomadSet      bool
	ConsulSet     bool
	VaultSet      bool
}

// EnvLookup is the env-reader seam. Production code uses OSEnvLookup; tests
// pass t.Setenv-managed values via the same hook (OSEnvLookup reads them).
type EnvLookup func(name string) (value string, present bool)

// OSEnvLookup reports whether the env var is present by delegating to
// os.LookupEnv. An env var set to the empty string is treated as present so
// that `NOMAD_VERSION=` triggers validation rather than a silent fallback.
func OSEnvLookup(name string) (string, bool) {
	return os.LookupEnv(name)
}

// ErrInvalid is returned for any validation failure produced by Resolve.
// Wrapped with %w so callers can errors.Is-test it.
var ErrInvalid = errors.New("invalid version override")

// Resolve applies the precedence rules and validates each effective value.
// On any validation failure it returns a single error that names the offending
// source (flag or env var) and the rejection reason.
//
// Precedence per package:
//   - if flag was provided (Set==true): use flag value, validate, ignore env.
//   - else if env var is present: use env value, validate.
//   - else: source=default, value="" (caller falls back to release defaults).
func Resolve(in FlagInputs, lookup EnvLookup) (Set, error) {
	var out Set

	type spec struct {
		flagName string
		envName  string
		flagVal  string
		flagSet  bool
		dst      *Resolved
	}

	specs := []spec{
		{FlagNomad, EnvNomad, in.NomadVersion, in.NomadSet, &out.Nomad},
		{FlagConsul, EnvConsul, in.ConsulVersion, in.ConsulSet, &out.Consul},
		{FlagVault, EnvVault, in.VaultVersion, in.VaultSet, &out.Vault},
	}

	for _, s := range specs {
		if s.flagSet {
			trimmed, err := validate(s.flagName, s.flagVal)
			if err != nil {
				return Set{}, err
			}
			*s.dst = Resolved{Value: trimmed, Source: SourceFlag}
			continue
		}
		envVal, present := lookup(s.envName)
		if !present {
			*s.dst = Resolved{Value: "", Source: SourceDefault}
			continue
		}
		trimmed, err := validate(s.envName, envVal)
		if err != nil {
			return Set{}, err
		}
		*s.dst = Resolved{Value: trimmed, Source: SourceEnv}
	}

	return out, nil
}

// validate applies the spec's validation rules to a single value and returns
// the trimmed canonical form. sourceName is included verbatim in errors so the
// caller's flag/env-var name appears in user-facing messages.
func validate(sourceName, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s: value must not be empty", ErrInvalid, sourceName)
	}
	if !orchestration.IsValidVersionToken(trimmed) {
		return "", fmt.Errorf("%w: %s: invalid format %q", ErrInvalid, sourceName, trimmed)
	}
	return trimmed, nil
}

// RenderLines returns the lines (without trailing newline) to be written to
// streams.ErrOut. Returns nil/empty when no package is overridden. Order is
// deterministic: nomad, consul, vault.
//
// Attribution:
//   - SourceFlag: print "  pkg: value" (no suffix; flag is already visible in
//     the user's invocation).
//   - SourceEnv:  print "  pkg: value (from env)".
//   - SourceDefault: omit entirely.
func RenderLines(r Set) []string {
	entries := []struct {
		name string
		ro   Resolved
	}{
		{"nomad", r.Nomad},
		{"consul", r.Consul},
		{"vault", r.Vault},
	}

	var body []string
	for _, e := range entries {
		if e.ro.Source == SourceDefault {
			continue
		}
		suffix := ""
		if e.ro.Source == SourceEnv {
			suffix = " (from env)"
		}
		body = append(body, fmt.Sprintf("  %s: %s%s", e.name, e.ro.Value, suffix))
	}
	if len(body) == 0 {
		return nil
	}
	lines := make([]string, 0, 1+len(body))
	lines = append(lines, "Version overrides:")
	lines = append(lines, body...)
	return lines
}
