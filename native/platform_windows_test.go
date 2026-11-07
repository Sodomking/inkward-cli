//go:build windows

package main

import (
	"strings"
	"testing"
)

func TestWindowsReadOnlyIntegration(t *testing.T) {
	// Real PowerShell and registry read. Never changes the runner proxy.
	_, err := readWindows(command)
	if err != nil {
		t.Fatal(err)
	}
	var script string
	_ = setWindows([]string{"<local>", "rimbaud.asia", "*.rimbaud.asia"}, false, func(_ string, a ...string) (string, error) { script = decodeScript(a); return "", nil })
	result, err := powershell(command, "$t=$null;$e=$null;[void][System.Management.Automation.Language.Parser]::ParseInput("+quotePS(script)+",[ref]$t,[ref]$e);if($e.Count){throw 'Invalid script'};'valid'")
	if err != nil || strings.TrimSpace(result) != "valid" {
		t.Fatal(result, err)
	}
}
