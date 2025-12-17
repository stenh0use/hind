package release

import (
	"errors"
	"testing"
)

// testData is a release data store with test data for use in tests.
var testData = New(
	"0.3.1",
	map[string]Info{
		"0.3.0": {
			Hind:       "0.3.0",
			Base:       "bullseye-slim",
			Consul:     "1.18.1",
			Nomad:      "1.7.6",
			Vault:      "1.15.4",
			Containerd: "1.6.31-1",
			DockerCe:   "26.0.1-1",
			CniPlugins: "1.3.0",
			Cilium:     "1.13.9",
		},
		"0.3.1": {
			Hind:       "0.3.1",
			Base:       "bullseye-slim",
			Consul:     "1.19.1",
			Nomad:      "1.8.1",
			Vault:      "1.16.1",
			Containerd: "1.6.31-1",
			DockerCe:   "26.0.1-1",
			CniPlugins: "1.3.0",
			Cilium:     "1.13.9",
		},
	},
)

func TestDefaultDataExists(t *testing.T) {
	// Verify versions store is properly initialized and can retrieve the latest release
	latest := versions.Latest()
	if latest.Hind == "" {
		t.Error("versions.Latest() returned empty Info")
	}
}

func TestData_Latest(t *testing.T) {
	got := testData.Latest()
	if got.Hind != "0.3.1" {
		t.Errorf("Latest().Hind = %q, want %q", got.Hind, "0.3.1")
	}
	if got.Consul != "1.19.1" {
		t.Errorf("Latest().Consul = %q, want %q", got.Consul, "1.19.1")
	}
}

func TestPackageLevelLatest(t *testing.T) {
	// Test that package-level Latest() function works
	got := Latest()
	if got.Hind == "" {
		t.Error("Latest() returned empty Info.Hind")
	}
}

func TestData_Get(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
		wantHind    string
		wantConsul  string
	}{
		{
			name:        "get existing version 0.3.0",
			version:     "0.3.0",
			expectError: false,
			wantHind:    "0.3.0",
			wantConsul:  "1.18.1",
		},
		{
			name:        "get existing version 0.3.1",
			version:     "0.3.1",
			expectError: false,
			wantHind:    "0.3.1",
			wantConsul:  "1.19.1",
		},
		{
			name:        "get non-existent version",
			version:     "999.0.0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testData.Get(tt.version)
			if tt.expectError {
				if err == nil {
					t.Errorf("Get(%q) want error, got nil", tt.version)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get(%q) unexpected error: %v", tt.version, err)
			}
			if got.Hind != tt.wantHind {
				t.Errorf("Get(%q).Hind = %q, want %q", tt.version, got.Hind, tt.wantHind)
			}
			if got.Consul != tt.wantConsul {
				t.Errorf("Get(%q).Consul = %q, want %q", tt.version, got.Consul, tt.wantConsul)
			}
		})
	}
}

func TestPackageLevelGet(t *testing.T) {
	// Test package-level Get() function works with versions data
	want := latest
	got, err := Get(want)
	if err != nil {
		t.Fatalf("Get(%q) unexpected error: %v", want, err)
	}
	if got.Hind != want {
		t.Errorf("Get(%q).Hind = %q, want %q", want, got.Hind, want)
	}
}

func TestData_List(t *testing.T) {
	list := testData.List()

	if len(list) != 2 {
		t.Errorf("List() returned %d releases, want 2", len(list))
	}

	// Check that both versions are present (order is not guaranteed)
	hasV030 := false
	hasV031 := false
	for _, v := range list {
		if v == "0.3.0" {
			hasV030 = true
		}
		if v == "0.3.1" {
			hasV031 = true
		}
	}

	if !hasV030 {
		t.Error("List() missing version 0.3.0")
	}
	if !hasV031 {
		t.Error("List() missing version 0.3.1")
	}
}

func TestPackageLevelList(t *testing.T) {
	// Test package-level List() function works with releases store
	list := List()
	if len(list) == 0 {
		t.Error("List() returned empty slice")
	}

	hasLatest := false

	for _, v := range list {
		if v == latest {
			hasLatest = true
		}
	}

	if !hasLatest {
		t.Errorf("List() missing latest version %q", latest)
	}
}

func TestInfo_GetPackage(t *testing.T) {
	rel := testData.Latest()

	tests := []struct {
		name        string
		packageName string
		expectError bool
		expected    string
	}{
		{
			name:        "get hind version",
			packageName: "hind",
			expectError: false,
			expected:    "0.3.1",
		},
		{
			name:        "get consul version",
			packageName: "consul",
			expectError: false,
			expected:    "1.19.1",
		},
		{
			name:        "get nomad version",
			packageName: "nomad",
			expectError: false,
			expected:    "1.8.1",
		},
		{
			name:        "get all packages",
			packageName: "vault",
			expectError: false,
			expected:    "1.16.1",
		},
		{
			name:        "get invalid package",
			packageName: "invalid",
			expectError: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rel.GetPackage(tt.packageName)
			if tt.expectError {
				if err == nil {
					t.Errorf("GetPackage(%q) want error, got nil", tt.packageName)
				} else if !errors.Is(err, ErrUnknownPackage) {
					t.Errorf("GetPackage(%q) error = %v, want ErrUnknownPackage", tt.packageName, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetPackage(%q) unexpected error: %v", tt.packageName, err)
			}
			if got != tt.expected {
				t.Errorf("GetPackage(%q) = %q, want %q", tt.packageName, got, tt.expected)
			}
		})
	}
}
