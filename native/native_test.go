package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestMain(m *testing.M) {
	// macOS exposes its temporary directory through /var -> /private/var.
	if d, e := filepath.EvalSymlinks(os.TempDir()); e == nil {
		os.Setenv("TMPDIR", d)
		os.Setenv("TMP", d)
		os.Setenv("TEMP", d)
	}
	os.Exit(m.Run())
}
func TestRoutesAndOptions(t *testing.T) {
	for _, args := range [][]string{{"--yes"}, {"open", "--exam", "english1", "--trial", "small"}, {"doctor", "--json"}} {
		if _, e := parse(args); e != nil {
			t.Fatal(e)
		}
	}
	for _, args := range [][]string{{"--exam", "https://evil"}, {"--trial", "bad"}, {"setup", "junk"}, {"--exam", "cet4", "--trial", "big"}} {
		if _, e := parse(args); e == nil {
			t.Fatal(args)
		}
	}
	o, _ := parse([]string{"open", "--exam", "english2", "--trial", "small"})
	u, _ := destination(o)
	if u != website+"#/brief?topic=postgraduate-english2-small-2026" {
		t.Fatal(u)
	}
}
func TestYAMLSemanticsAndUndo(t *testing.T) {
	before := []byte("# original comment\nprepend:\n  - DOMAIN,other.example,PROXY # keep\nappend: [MATCH,PROXY]\ndelete: []\n")
	after, changed, e := editYAML(before, false)
	if e != nil || !changed {
		t.Fatal(e)
	}
	if !strings.Contains(string(after), "original comment") || !strings.Contains(string(after), "# keep") {
		t.Fatal(string(after))
	}
	n, _ := doc(after)
	if field(n, "prepend").Content[0].Value != directRule {
		t.Fatal("not first")
	}
	again, ch, e := editYAML(after, false)
	if e != nil || ch || string(again) != string(after) {
		t.Fatal("not idempotent")
	}
	restored, ch, e := editYAML(after, true)
	if e != nil || !ch {
		t.Fatal(e)
	}
	var a, b any
	na, _ := doc(before)
	nb, _ := doc(restored)
	na.Decode(&a)
	nb.Decode(&b)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("undo lost data")
	}
}
func TestYAMLRejectUnsafeAndPreserveUserRule(t *testing.T) {
	for _, s := range []string{"prepend: []\nprepend: []\n", "prepend: []\n---\nprepend: []", "prepend: test", "prepend: [true]", "prepend: &r []\nother: *r\n", "prepend:\n  # INKWARD direct rule v2\n  - MATCH,PROXY\n"} {
		_, _, e := editYAML([]byte(s), false)
		if e == nil {
			t.Fatalf("accepted %s", s)
		}
	}
	s := []byte("prepend: [DOMAIN-SUFFIX\\,rimbaud.asia\\,DIRECT]\n")
	_ = s
	before := []byte("prepend:\n  - " + directRule + "\n")
	after, ch, e := editYAML(before, false)
	if e != nil || ch {
		t.Fatal(e)
	}
	restored, ch, e := editYAML(after, true)
	if e != nil || ch || string(restored) != string(before) {
		t.Fatal("removed user's rule")
	}
}
func TestRulePathUsesCurrentProfile(t *testing.T) {
	d := t.TempDir()
	os.Mkdir(filepath.Join(d, "profiles"), 0700)
	os.WriteFile(filepath.Join(d, "profiles", "r.yaml"), []byte("prepend: []\n"), 0600)
	profiles := "current: active\nitems:\n  - uid: inactive\n    option: {rules: not-found}\n  - uid: active\n    option: {rules: rules1}\n  - uid: rules1\n    type: rules\n    file: r.yaml\n"
	os.WriteFile(filepath.Join(d, "profiles.yaml"), []byte(profiles), 0600)
	p, e := vergeRulePath(d)
	if e != nil || p != filepath.Join(d, "profiles", "r.yaml") {
		t.Fatal(p, e)
	}
	os.WriteFile(filepath.Join(d, "profiles.yaml"), []byte(strings.Replace(profiles, "file: r.yaml", "file: ../escape.yaml", 1)), 0600)
	if _, e := vergeRulePath(d); e == nil {
		t.Fatal("traversal")
	}
}
func TestBackupAndUndoPreserveLaterRule(t *testing.T) {
	d := t.TempDir()
	file := filepath.Join(d, "rules.yaml")
	original := []byte("prepend: []\nappend: []\ndelete: []\n")
	os.WriteFile(file, original, 0600)
	state := filepath.Join(d, "state")
	if s := applyYAML(file, state, false); s.Status != "planned" {
		t.Fatal(s)
	}
	b, _ := os.ReadFile(file)
	if string(b) != string(original) {
		t.Fatal("dry write")
	}
	s := applyYAML(file, state, true)
	if s.Status != "needs-reload" {
		t.Fatal(s)
	}
	st, e := loadState(state)
	if e != nil || len(st.Changes) != 1 {
		t.Fatal(e)
	}
	backup, _ := os.ReadFile(st.Changes[0].Backup)
	if string(backup) != string(original) {
		t.Fatal("backup")
	}
	b, _ = os.ReadFile(file)
	b = append(b, []byte("later: keep\n")...)
	os.WriteFile(file, b, 0600)
	result := undo(state, func(string, ...string) (string, error) { t.Fatal("unexpected system command"); return "", nil })
	if result[0].Status != "needs-reload" {
		t.Fatal(result)
	}
	b, _ = os.ReadFile(file)
	if !strings.Contains(string(b), "later: keep") || strings.Contains(string(b), directRule) {
		t.Fatal(string(b))
	}
}
func TestLockAndConcurrentChange(t *testing.T) {
	d := t.TempDir()
	unlock, e := lockState(d)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := lockState(d); e == nil {
		t.Fatal("second lock")
	}
	unlock()
	f := filepath.Join(d, "x")
	os.WriteFile(f, []byte("new"), 0600)
	if e := replaceChecked(f, []byte("old"), []byte("oops")); e == nil {
		t.Fatal("clobbered")
	}
}
func TestDomains(t *testing.T) {
	before := []string{"<local>", "*.company.local", "rimbaud.asia"}
	after, added := mergeDomains(before)
	if !reflect.DeepEqual(added, []string{"*.rimbaud.asia"}) {
		t.Fatal(added)
	}
	after = append(after, "new.example")
	restored := removeDomains(after, added)
	if !reflect.DeepEqual(restored, append(before, "new.example")) {
		t.Fatal(restored)
	}
	if !reflect.DeepEqual(identifyClients("/Applications/Clash Verge.app/Contents/MacOS/clash-verge\nverge-mihomo\nsomething"), []string{"Clash Verge", "Mihomo"}) {
		t.Fatal("detection")
	}
}
func TestMacProxyIsScopedAndIdempotent(t *testing.T) {
	value := "*.company.local\n"
	writes := 0
	run := func(_ string, args ...string) (string, error) {
		switch args[0] {
		case "-listallnetworkservices":
			return "An asterisk (*) denotes disabled services.\nWi-Fi\n*Disabled\n", nil
		case "-getwebproxy", "-getsecurewebproxy":
			return "Enabled: Yes\n", nil
		case "-getsocksfirewallproxy", "-getautoproxyurl", "-getproxyautodiscovery":
			return "Enabled: No\n", nil
		case "-getproxybypassdomains":
			return value, nil
		case "-setproxybypassdomains":
			writes++
			value = strings.Join(args[2:], "\n") + "\n"
			return "", nil
		}
		return "", errors.New("unexpected")
	}
	d := t.TempDir()
	s := macBypass(d, false, run)
	if s[0].Status != "planned" || writes != 0 {
		t.Fatal(s)
	}
	s = macBypass(d, true, run)
	if s[0].Status != "settings-verified" || writes != 1 || !strings.Contains(value, "*.company.local") {
		t.Fatal(s, value)
	}
	s = macBypass(d, true, run)
	if s[0].Status != "already-configured" || writes != 1 {
		t.Fatal(s)
	}
}
func TestMacPACDoesNotWrite(t *testing.T) {
	run := func(_ string, a ...string) (string, error) {
		switch a[0] {
		case "-listallnetworkservices":
			return "Wi-Fi", nil
		case "-getautoproxyurl":
			return "Enabled: Yes", nil
		case "-getwebproxy", "-getsecurewebproxy", "-getsocksfirewallproxy", "-getproxyautodiscovery":
			return "Enabled: No", nil
		}
		t.Fatal("unexpected write")
		return "", nil
	}
	s := macBypass(t.TempDir(), true, run)
	if s[0].Status != "needs-agent" {
		t.Fatal(s)
	}
}
func decodeScript(args []string) string {
	b, _ := base64.StdEncoding.DecodeString(args[len(args)-1])
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u))
}
func TestWindowsScopeAndPolicies(t *testing.T) {
	p := WinProxy{Enabled: true, Exists: true, Kind: "String", Value: "<local>;*.company.local"}
	writes := 0
	run := func(_ string, a ...string) (string, error) {
		s := decodeScript(a)
		if strings.Contains(s, "ConvertTo-Json") {
			b, _ := json.Marshal(p)
			return string(b), nil
		}
		writes++
		if !strings.Contains(s, "ProxyOverride") || strings.Contains(s, "Set-ItemProperty") || !strings.Contains(s, "<local>;*.company.local;rimbaud.asia;*.rimbaud.asia") {
			t.Fatal(s)
		}
		p.Value += ";rimbaud.asia;*.rimbaud.asia"
		return "", nil
	}
	s := windowsBypass(t.TempDir(), true, run)
	if s[0].Status != "settings-verified" || writes != 1 {
		t.Fatal(s)
	}
	p.Managed = true
	s = windowsBypass(t.TempDir(), true, run)
	if s[0].Status != "needs-agent" || writes != 1 {
		t.Fatal(s)
	}
	p.Managed = false
	p.Automatic = true
	s = windowsBypass(t.TempDir(), true, run)
	if s[0].Status != "needs-agent" || writes != 1 {
		t.Fatal(s)
	}
}
func TestStateMergesAddedWithoutLosingOriginal(t *testing.T) {
	d := t.TempDir()
	record(d, Change{Kind: "windows-system", Added: []string{"rimbaud.asia"}})
	record(d, Change{Kind: "windows-system", Added: []string{"*.rimbaud.asia"}, BeforeExists: true})
	s, e := loadState(d)
	if e != nil || len(s.Changes) != 1 || len(s.Changes[0].Added) != 2 || s.Changes[0].BeforeExists {
		t.Fatal(s, e)
	}
}
