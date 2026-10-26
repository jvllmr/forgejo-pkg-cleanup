//go:build e2e

// End-to-end tests against a local Forgejo instance.
// Start it with test-env/run.sh, which also runs these tests.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	forgejoURL = "http://127.0.0.1:3000"
	owner      = "testuser"
	password   = "testpassword"

	keepVersions = `^(?:v)?(?:\d+)(?:\.(\d+))?(?:\.(\d+))?(?:[a|b]\d+)?$`
	pkgRegex     = `^testpkg$`
	pkgTypeRegex = `^generic$`

	// retry settings for waiting on the forgejo stack
	waitAttempts = 30
	waitInterval = 2 * time.Second
)

var binary string // action binary built in TestMain

// fixtures seeded before each scenario; "want" is the package set after a run
// with retention 0s.
var fixtures = []struct{ Name, Version string }{
	{"testpkg", "1.0.0"},       // matches keepVersions -> kept
	{"testpkg", "1.2.3"},       // matches keepVersions -> kept
	{"testpkg", "nightly-abc"}, // no match -> deleted outside retention
	{"otherpkg", "999-dev"},    // no match, but excluded by package regex -> kept
}

type forgejoPackage struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (p forgejoPackage) String() string {
	return fmt.Sprintf("%s %s %s", p.Type, p.Name, p.Version)
}

func apiRequest(t *testing.T, method, path string, body []byte) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(method, forgejoURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(owner, password)
	if body != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	return http.DefaultClient.Do(req)
}

func listPackages(t *testing.T) []forgejoPackage {
	t.Helper()
	resp, err := apiRequest(t, http.MethodGet, "/api/v1/packages/"+owner, nil)
	if err != nil {
		t.Fatalf("list packages: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list packages: status %d", resp.StatusCode)
	}
	var packages []forgejoPackage
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		t.Fatalf("decode package list: %v", err)
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].String() < packages[j].String() })
	return packages
}

func deleteAllPackages(t *testing.T) {
	t.Helper()
	for _, p := range listPackages(t) {
		resp, err := apiRequest(t, http.MethodDelete,
			fmt.Sprintf("/api/v1/packages/%s/%s/%s/%s", owner, p.Type, p.Name, p.Version), nil)
		if err != nil {
			t.Fatalf("delete package %s: %v", p, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("delete package %s: status %d", p, resp.StatusCode)
		}
	}
}

func uploadGeneric(t *testing.T, name, version string) {
	t.Helper()
	resp, err := apiRequest(t, http.MethodPut,
		fmt.Sprintf("/api/packages/%s/generic/%s/%s/file.txt", owner, name, version),
		[]byte("test"))
	if err != nil {
		t.Fatalf("upload %s %s: %v", name, version, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload %s %s: status %d", name, version, resp.StatusCode)
	}
}

func seed(t *testing.T) {
	t.Helper()
	deleteAllPackages(t)
	for _, f := range fixtures {
		uploadGeneric(t, f.Name, f.Version)
	}
	// give forgejo time to persist created_at, so retention 0s sees non-zero age
	time.Sleep(time.Second)
}

// runAction runs the built action binary the same way forgejo-runner would:
// via INPUT_* environment variables.
func runAction(t *testing.T, retention string) {
	t.Helper()
	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"INPUT_INSTANCE="+forgejoURL,
		"INPUT_USERNAME="+owner,
		"INPUT_PASSWORD="+password,
		"INPUT_PACKAGE="+pkgRegex,
		"INPUT_PACKAGETYPE="+pkgTypeRegex,
		"INPUT_OWNER="+owner,
		"INPUT_KEEPVERSIONS="+keepVersions,
		"INPUT_RETENTION="+retention,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("action run (retention %s) failed: %v\n%s", retention, err, out.String())
	}
}

func assertPackages(t *testing.T, want []forgejoPackage) {
	t.Helper()
	got := listPackages(t)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected package set:\n got:  %s\n want: %s",
			strings.Join(mapString(got), ", "), strings.Join(mapString(want), ", "))
	}
}

func mapString(ps []forgejoPackage) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	return out
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "forgejo-pkg-cleanup-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	binary = filepath.Join(dir, "forgejo-pkg-cleanup")

	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "go build failed: %v\n%s", err, out)
		os.Exit(1)
	}

	// wait for the forgejo stack started by test-env/run.sh
	for i := 0; i < waitAttempts; i++ {
		resp, err := http.Get(forgejoURL + "/api/healthz")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			os.Exit(m.Run())
		}
		if err == nil {
			resp.Body.Close()
		}
		time.Sleep(waitInterval)
	}
	fmt.Fprintf(
		os.Stderr,
		"forgejo at %s not reachable; start it with test-env/run.sh\n",
		forgejoURL,
	)
	os.Exit(1)
}

func TestCleanupDeletesNonMatchingVersions(t *testing.T) {
	seed(t)
	runAction(t, "0s")
	assertPackages(t, []forgejoPackage{
		{Type: "generic", Name: "otherpkg", Version: "999-dev"},
		{Type: "generic", Name: "testpkg", Version: "1.0.0"},
		{Type: "generic", Name: "testpkg", Version: "1.2.3"},
	})
}

func TestRetentionKeepsFreshPackages(t *testing.T) {
	seed(t)
	runAction(t, "1h")
	assertPackages(t, []forgejoPackage{
		{Type: "generic", Name: "otherpkg", Version: "999-dev"},
		{Type: "generic", Name: "testpkg", Version: "1.0.0"},
		{Type: "generic", Name: "testpkg", Version: "1.2.3"},
		{Type: "generic", Name: "testpkg", Version: "nightly-abc"},
	})
}

func TestRetentionDaysSuffixKeepsFreshPackages(t *testing.T) {
	seed(t)
	// "2d" was rejected by time.ParseDuration before ParseRetention existed
	runAction(t, "2d")
	assertPackages(t, []forgejoPackage{
		{Type: "generic", Name: "otherpkg", Version: "999-dev"},
		{Type: "generic", Name: "testpkg", Version: "1.0.0"},
		{Type: "generic", Name: "testpkg", Version: "1.2.3"},
		{Type: "generic", Name: "testpkg", Version: "nightly-abc"},
	})
}

func TestInvalidRetentionFailsWithoutDeleting(t *testing.T) {
	seed(t)
	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"INPUT_INSTANCE="+forgejoURL,
		"INPUT_USERNAME="+owner,
		"INPUT_PASSWORD="+password,
		"INPUT_PACKAGE="+pkgRegex,
		"INPUT_PACKAGETYPE="+pkgTypeRegex,
		"INPUT_OWNER="+owner,
		"INPUT_KEEPVERSIONS="+keepVersions,
		"INPUT_RETENTION=banana",
	)
	if err := cmd.Run(); err == nil {
		t.Fatal("action with invalid retention should fail")
	}
	assertPackages(t, []forgejoPackage{
		{Type: "generic", Name: "otherpkg", Version: "999-dev"},
		{Type: "generic", Name: "testpkg", Version: "1.0.0"},
		{Type: "generic", Name: "testpkg", Version: "1.2.3"},
		{Type: "generic", Name: "testpkg", Version: "nightly-abc"},
	})
}
