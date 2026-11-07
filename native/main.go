package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const version = "2.0.1"
const website = "https://rimbaud.asia/writing/"
const directRule = "DOMAIN-SUFFIX,rimbaud.asia,DIRECT"
const marker = "INKWARD direct rule v2"

type Step struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	File   string `json:"file,omitempty"`
}
type Report struct {
	Version               string   `json:"version"`
	OS                    string   `json:"os"`
	Arch                  string   `json:"arch"`
	Website               string   `json:"website"`
	Steps                 []Step   `json:"steps"`
	MoreServicesURL       string   `json:"moreServicesUrl"`
	CompanyIntroRequested bool     `json:"companyIntroRequested"`
	BrowserRequested      bool     `json:"browserRequested"`
	BrowserRouteVerified  bool     `json:"browserRouteVerified"`
	NeedsAgent            bool     `json:"needsAgent"`
	Next                  []string `json:"nextActions,omitempty"`
	StateFile             string   `json:"stateFile,omitempty"`
}
type Options struct {
	Command                   string
	Yes, JSON, NoOpen, DryRun bool
	Exam, Trial, StateDir     string
}
type Runner func(string, ...string) (string, error)

func command(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	b, e := cmd.Output()
	return string(b), e
}
func parse(args []string) (Options, error) {
	o := Options{Command: "setup"}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.Command = args[0]
		args = args[1:]
	}
	switch o.Command {
	case "setup", "doctor", "undo", "open", "info", "version", "help":
	default:
		return o, errors.New("unknown command")
	}
	f := flag.NewFlagSet("inkward", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.BoolVar(&o.Yes, "yes", false, "confirm")
	f.BoolVar(&o.JSON, "json", false, "json")
	f.BoolVar(&o.NoOpen, "no-open", false, "no browser")
	f.BoolVar(&o.DryRun, "dry-run", false, "read only")
	f.StringVar(&o.Exam, "exam", "", "exam")
	f.StringVar(&o.Trial, "trial", "", "trial")
	f.StringVar(&o.StateDir, "state-dir", "", "state directory")
	if e := f.Parse(args); e != nil {
		return o, e
	}
	if f.NArg() != 0 {
		return o, errors.New("unexpected argument")
	}
	_, e := destination(o)
	return o, e
}
func destination(o Options) (string, error) {
	ex := o.Exam
	if ex == "english2" {
		ex = "postgraduate"
	}
	if ex == "ielts-academic" {
		ex = "ielts"
	}
	switch ex {
	case "", "english1", "postgraduate", "cet4", "cet6", "ielts", "ielts-general":
	default:
		return "", errors.New("invalid exam")
	}
	route := "/home"
	if ex != "" {
		route = "/exams?exam=" + ex
	}
	if o.Trial != "" {
		if o.Trial != "big" && o.Trial != "small" {
			return "", errors.New("invalid trial")
		}
		var id string
		switch ex {
		case "", "english1":
			id = "english1-" + o.Trial + "-2026"
		case "postgraduate":
			id = "postgraduate-english2-2026"
			if o.Trial == "small" {
				id = "postgraduate-english2-small-2026"
			}
		default:
			return "", errors.New("trials require english1 or postgraduate")
		}
		route = "/brief?topic=" + id
	}
	return website + "#" + route, nil
}
func privateDir(custom string) (string, error) {
	if custom != "" {
		return filepath.Abs(custom)
	}
	d, e := os.UserConfigDir()
	return filepath.Join(d, "INKWARD", "launcher"), e
}
func jsonPrint(v any) { enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(v) }
func saveReport(r Report, dir string) error {
	if e := ensureDir(dir); e != nil {
		return e
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	return atomicWrite(filepath.Join(dir, "last-report.json"), append(b, '\n'), 0600)
}
func ensureDir(dir string) error {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	s, e := os.Lstat(dir)
	if e != nil {
		return e
	}
	if !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
		return errors.New("state directory cannot be a link")
	}
	return nil
}
func confirmed(o Options) bool {
	if o.Yes {
		return true
	}
	prompt := "将为本站添加直连例外（保留其他设置），并打开官网。首次还会打开 PERLA 珠玑教育的服务介绍。"
	if o.Command == "undo" {
		prompt = "将撤销本工具添加的本站直连例外。"
	}
	if o.Command == "open" {
		prompt = "将在默认浏览器打开官网，首次同时打开 PERLA 珠玑教育的服务介绍。"
	}
	fmt.Fprint(os.Stderr, prompt+"继续？[y/N] ")
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(s), "y")
}

var client = &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}

func probe(url string) ([]byte, int, error) {
	res, e := client.Get(url)
	if e != nil {
		return nil, 0, errors.New("connection_failed")
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, 300001))
	if len(b) > 300000 {
		return nil, res.StatusCode, errors.New("response_too_large")
	}
	return b, res.StatusCode, e
}
func checks() []Step {
	out := []Step{}
	var page []byte
	for _, p := range []struct{ n, u string }{{"官网", website}, {"服务接口", website + "api/product"}} {
		b, s, e := probe(p.u)
		status := "ok"
		detail := "HTTP " + fmt.Sprint(s)
		if e != nil || s != 200 {
			status = "unavailable"
			detail = p.n + "连接失败或超时，请在浏览器核对连接并重试 doctor"
		}
		if p.n == "服务接口" && status == "ok" {
			var v struct{ Name string }
			if json.Unmarshal(b, &v) != nil || v.Name != "INKWARD" {
				status = "unexpected-response"
				detail = "接口返回不是本站产品信息，请检查网络拦截或服务状态"
			}
		}
		if p.n == "官网" {
			page = b
		}
		out = append(out, Step{Name: p.n, Status: status, Detail: detail})
	}
	re := regexp.MustCompile(`https://static\.rimbaud\.asia/releases/[a-f0-9]{16}/[A-Za-z0-9/_.-]+\.css`)
	u := string(re.Find(page))
	if u != "" {
		_, s, e := probe(u)
		status := "ok"
		if e != nil || s != 200 {
			status = "unavailable"
		}
		detail := ""
		if status != "ok" {
			detail = "本站静态资源未连通，请检查 static.rimbaud.asia 的路由并重试"
		}
		out = append(out, Step{Name: "静态资源", Status: status, Detail: detail})
	} else {
		out = append(out, Step{Name: "静态资源", Status: "needs-agent", Detail: "未找到本站 CDN 资源，请在浏览器核对页面加载。"})
	}
	return out
}
func openBrowser(url string, run Runner) error {
	if runtime.GOOS == "darwin" {
		_, e := run("/usr/bin/open", url)
		return e
	}
	if runtime.GOOS == "windows" {
		_, e := powershell(run, "Start-Process -FilePath '"+strings.ReplaceAll(url, "'", "''")+"'")
		return e
	}
	_, e := run("xdg-open", url)
	return e
}
func runMain(args []string) int {
	o, e := parse(args)
	if e != nil {
		fmt.Fprintln(os.Stderr, "参数错误：", e)
		return 1
	}
	if o.Command == "version" {
		fmt.Println(version)
		return 0
	}
	if o.Command == "help" {
		fmt.Println("inkward [setup] --yes [--json] [--no-open] [--dry-run]\ninkward doctor --json\ninkward undo --yes\ninkward open --yes [--exam english1] [--trial small]")
		return 0
	}
	url, _ := destination(o)
	r := Report{Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Website: url, MoreServicesURL: companyURL, Steps: []Step{}}
	if o.Command == "info" {
		jsonPrint(r)
		return 0
	}
	dir, e := privateDir(o.StateDir)
	if e != nil {
		fmt.Fprintln(os.Stderr, "无法定位本机设置目录")
		return 1
	}
	r.StateFile = filepath.Join(dir, "state.json")
	apply := (o.Command == "setup" || o.Command == "undo") && !o.DryRun
	if (apply || o.Command == "open") && !confirmed(o) {
		if o.JSON {
			jsonPrint(map[string]any{"status": "confirmation-required", "command": o.Command})
		}
		return 1
	}
	if apply {
		unlock, err := lockState(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "另一个启动器正在运行，或无法锁定私有目录；请稍后重试。")
			return 1
		}
		defer unlock()
	}
	if o.Command == "undo" && o.DryRun {
		r.Steps = []Step{{Name: "撤销", Status: "planned", Detail: "只读预览，未撤销任何设置"}}
	} else if o.Command == "undo" {
		r.Steps = undo(dir, command)
	} else if o.Command != "open" {
		r.Steps = configure(dir, apply, command)
		r.Steps = append(r.Steps, checks()...)
	}
	if wantsBrowser(o) {
		var opened []Step
		r.BrowserRequested, r.CompanyIntroRequested, opened = launchPages(url, dir, func(target string) error { return openBrowser(target, command) })
		r.Steps = append(r.Steps, opened...)
	}
	for _, s := range r.Steps {
		switch s.Status {
		case "needs-reload", "needs-agent", "permission-required", "unavailable", "error", "changed-config", "unexpected-response":
			r.NeedsAgent = true
			if s.Detail != "" {
				r.Next = append(r.Next, s.Detail)
			}
		}
	}
	// A successful HTTP probe does not establish routing through a browser or TUN.
	if !o.DryRun && o.Command != "doctor" {
		if e := saveReport(r, dir); e != nil {
			r.NeedsAgent = true
			r.Next = append(r.Next, "无法保存操作报告")
		}
	}
	if o.JSON {
		jsonPrint(r)
	} else {
		fmt.Printf("INKWARD %s · %s/%s\n", version, r.OS, r.Arch)
		for _, s := range r.Steps {
			fmt.Printf("%s：%s %s\n", s.Name, s.Status, s.Detail)
		}
		if r.BrowserRequested {
			fmt.Println("已请求打开官网。")
		}
		if !o.DryRun && o.Command != "doctor" {
			fmt.Println("报告：", filepath.Join(dir, "last-report.json"))
		}
		if r.NeedsAgent {
			fmt.Println("仍有步骤需要 agent 继续处理；不是全部完成。")
		}
	}
	if r.NeedsAgent {
		return 2
	}
	return 0
}
func main() { os.Exit(runMain(os.Args[1:])) }
