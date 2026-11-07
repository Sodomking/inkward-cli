package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestFirstLaunchIntroducesCompanyOnlyOnce(t *testing.T) {
	d := t.TempDir()
	var opened []string
	open := func(u string) error { opened = append(opened, u); return nil }
	training, intro, steps := launchPages(website+"#/home", d, open)
	if !training || !intro || len(steps) != 1 {
		t.Fatal(training, intro, steps)
	}
	if !reflect.DeepEqual(opened, []string{website + "#/home", companyURL}) {
		t.Fatal(opened)
	}
	opened = nil
	training, intro, _ = launchPages(website+"#/exams?exam=cet4", d, open)
	if !training || intro || !reflect.DeepEqual(opened, []string{website + "#/exams?exam=cet4"}) {
		t.Fatal(opened)
	}
	b, e := os.ReadFile(filepath.Join(d, introductionFile))
	if e != nil || len(b) == 0 {
		t.Fatal(e)
	}
}
func TestFirstLaunchDoesNotConsumeIntroWhenTrainingFails(t *testing.T) {
	d := t.TempDir()
	_, intro, _ := launchPages(website, d, func(string) error { return errors.New("rejected") })
	if intro {
		t.Fatal("unexpected introduction")
	}
	if _, e := os.Stat(filepath.Join(d, introductionFile)); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
func TestRejectedCompanyOpenCanRetry(t *testing.T) {
	d := t.TempDir()
	training, intro, _ := launchPages(website, d, func(u string) error {
		if u == companyURL {
			return errors.New("rejected")
		}
		return nil
	})
	if !training || intro {
		t.Fatal(training, intro)
	}
	if _, e := os.Stat(filepath.Join(d, introductionFile)); !os.IsNotExist(e) {
		t.Fatal(e)
	}
	_, intro, _ = launchPages(website, d, func(string) error { return nil })
	if !intro {
		t.Fatal("retry was lost")
	}
}
func TestConcurrentLaunchesDoNotRepeatCompany(t *testing.T) {
	d := t.TempDir()
	var count atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			launchPages(website, d, func(u string) error {
				if u == companyURL {
					count.Add(1)
				}
				return nil
			})
		}()
	}
	wg.Wait()
	if count.Load() != 1 {
		t.Fatal(count.Load())
	}
}
func TestReadOnlyAndNoOpenCommandsNeverRequestPages(t *testing.T) {
	for _, o := range []Options{{Command: "doctor"}, {Command: "info"}, {Command: "undo"}, {Command: "setup", DryRun: true}, {Command: "open", DryRun: true}, {Command: "setup", NoOpen: true}, {Command: "open", NoOpen: true}} {
		if wantsBrowser(o) {
			t.Fatal(o)
		}
	}
	for _, name := range []string{"open", "setup"} {
		if !wantsBrowser(Options{Command: name}) {
			t.Fatal(name)
		}
	}
}
