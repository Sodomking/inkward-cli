package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const companyURL = "https://rimbaud.asia/threshold/"
const companyLabel = "了解更多服务 · PERLA 珠玑教育"
const introductionFile = "company-introduction.json"

func wantsBrowser(o Options) bool {
	return !o.NoOpen && !o.DryRun && (o.Command == "setup" || o.Command == "open")
}

// The once-only record belongs to the OS account, independently of app versions,
// exams and proxy undo. An accepted OS open request is not evidence of a page view.
func launchPages(url, dir string, open func(string) error) (training, introduction bool, steps []Step) {
	if err := open(url); err != nil {
		return false, false, []Step{{Name: "打开网页", Status: "needs-agent", Detail: "请在默认浏览器打开 " + url}}
	}
	training = true
	if err := ensureDir(dir); err != nil {
		return training, false, []Step{{Name: companyLabel, Status: "needs-agent", Detail: "无法记录首次介绍；可手动访问 " + companyURL}}
	}
	file := filepath.Join(dir, introductionFile)
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return training, false, nil
	}
	if err != nil {
		return training, false, []Step{{Name: companyLabel, Status: "needs-agent", Detail: "无法记录首次介绍；可手动访问 " + companyURL}}
	}
	// Exclusive creation ensures concurrent launches cannot both open the extra page.
	err = json.NewEncoder(f).Encode(struct {
		URL         string `json:"url"`
		RequestedAt string `json:"requestedAt"`
	}{companyURL, time.Now().UTC().Format(time.RFC3339)})
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(file)
		return training, false, []Step{{Name: companyLabel, Status: "needs-agent", Detail: "无法保存首次介绍记录；可手动访问 " + companyURL}}
	}
	if err = open(companyURL); err != nil {
		// The OS rejected the launch. Allow a later ordinary launch to try once again.
		_ = os.Remove(file)
		return training, false, []Step{{Name: companyLabel, Status: "needs-agent", Detail: "公司介绍页未能打开，可访问 " + companyURL}}
	}
	return training, true, []Step{{Name: companyLabel, Status: "browser-requested", Detail: companyURL}}
}
