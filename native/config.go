package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Change struct {
	Kind         string   `json:"kind"`
	File         string   `json:"file,omitempty"`
	Service      string   `json:"service,omitempty"`
	Added        []string `json:"added,omitempty"`
	Backup       string   `json:"backup,omitempty"`
	BeforeExists bool     `json:"beforeExists,omitempty"`
	BeforeHash   string   `json:"beforeHash,omitempty"`
}
type State struct {
	Version int      `json:"version"`
	Changes []Change `json:"changes"`
}

func readSafe(file string) ([]byte, error) {
	st, e := os.Lstat(file)
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Size() > 2*1024*1024 {
		return nil, errors.New("not a regular bounded file")
	}
	resolved, e := filepath.EvalSymlinks(file)
	if e != nil {
		return nil, e
	}
	abs, e := filepath.Abs(file)
	if e != nil || resolved != abs {
		return nil, errors.New("linked path not supported")
	}
	return os.ReadFile(file)
}
func atomicWrite(file string, data []byte, mode os.FileMode) error {
	f, e := os.CreateTemp(filepath.Dir(file), ".inkward-*")
	if e != nil {
		return e
	}
	temp := f.Name()
	defer os.Remove(temp)
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(temp, file)
}
func loadState(dir string) (State, error) {
	s := State{Version: 2}
	b, e := readSafe(filepath.Join(dir, "state.json"))
	if os.IsNotExist(e) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	e = json.Unmarshal(b, &s)
	if s.Version != 2 {
		return s, errors.New("unsupported state")
	}
	return s, e
}
func record(dir string, c Change) error {
	if e := ensureDir(dir); e != nil {
		return e
	}
	s, e := loadState(dir)
	if e != nil {
		return e
	}
	found := false
	for i, x := range s.Changes {
		if x.Kind == c.Kind && x.File == c.File && x.Service == c.Service {
			found = true
			for _, a := range c.Added {
				if !contains(s.Changes[i].Added, a) {
					s.Changes[i].Added = append(s.Changes[i].Added, a)
				}
			}
		}
	}
	if !found {
		s.Changes = append(s.Changes, c)
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return atomicWrite(filepath.Join(dir, "state.json"), b, 0600)
}
func doc(data []byte) (*yaml.Node, error) {
	var n yaml.Node
	d := yaml.NewDecoder(bytes.NewReader(data))
	if e := d.Decode(&n); e != nil {
		return nil, e
	}
	var extra yaml.Node
	if e := d.Decode(&extra); e != io.EOF {
		return nil, errors.New("multiple YAML documents")
	}
	if len(n.Content) != 1 || n.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("expected YAML mapping")
	}
	if hasAlias(&n) {
		return nil, errors.New("YAML aliases or anchors unsupported")
	}
	var validate any
	if e := n.Decode(&validate); e != nil {
		return nil, e
	}
	return &n, nil
}
func field(n *yaml.Node, key string) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.DocumentNode {
		return field(n.Content[0], key)
	}
	if n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}
func scalar(n *yaml.Node, key string) string {
	x := field(n, key)
	if x != nil && x.Kind == yaml.ScalarNode {
		return x.Value
	}
	return ""
}
func editYAML(data []byte, remove bool) ([]byte, bool, error) {
	n, e := doc(data)
	if e != nil {
		return nil, false, e
	}
	seq := field(n, "prepend")
	if seq == nil || seq.Kind != yaml.SequenceNode || seq.Alias != nil {
		return nil, false, errors.New("expected prepend list")
	}
	for _, x := range seq.Content {
		if x.Kind != yaml.ScalarNode || x.Tag != "!!str" {
			return nil, false, errors.New("complex rule unsupported")
		}
	}
	index := -1
	for i, x := range seq.Content {
		if strings.Contains(x.HeadComment, marker) {
			if x.Value != directRule || index != -1 {
				return nil, false, errors.New("managed rule was changed")
			}
			index = i
		}
	}
	if remove {
		if index < 0 {
			return data, false, nil
		}
		seq.Content = append(seq.Content[:index], seq.Content[index+1:]...)
	} else {
		if index > 0 {
			return nil, false, errors.New("managed rule priority changed")
		}
		if index == 0 {
			return data, false, nil
		}
		if len(seq.Content) > 0 && seq.Content[0].Value == directRule {
			return data, false, nil
		}
		seq.Content = append([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: directRule, HeadComment: marker}}, seq.Content...)
		seq.Style = 0
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if e = enc.Encode(n); e != nil {
		return nil, false, e
	}
	_ = enc.Close()
	return b.Bytes(), true, nil
}
func replaceChecked(file string, before, after []byte) error {
	current, e := readSafe(file)
	if e != nil {
		return e
	}
	if !bytes.Equal(current, before) {
		return errors.New("file changed concurrently")
	}
	st, e := os.Stat(file)
	if e != nil {
		return e
	}
	return atomicWrite(file, after, st.Mode().Perm())
}
func applyYAML(file, dir string, apply bool) Step {
	s := Step{Name: "Clash Verge 本站直连", File: file}
	before, e := readSafe(file)
	if e != nil {
		s.Status = "needs-agent"
		s.Detail = "无法读取当前规则扩展，请在客户端的编辑规则中添加 " + directRule
		return s
	}
	after, changed, e := editYAML(before, false)
	if e != nil {
		s.Status = "needs-agent"
		s.Detail = "当前规则扩展格式无法自动处理，请在客户端前置添加 " + directRule
		return s
	}
	s.Status = "needs-reload"
	s.Detail = "在 Clash Verge 重载当前配置，确认 Rule 模式及本站 DIRECT 规则；然后检查浏览器。"
	if !apply {
		s.Status = "planned"
		return s
	}
	if !changed {
		return s
	}
	if e = ensureDir(dir); e != nil {
		s.Status = "error"
		s.Detail = "无法建立私有备份目录"
		return s
	}
	backup := filepath.Join(dir, fmt.Sprintf("rules-%d.bak", time.Now().UnixNano()))
	if e = os.WriteFile(backup, before, 0600); e != nil {
		s.Status = "error"
		s.Detail = "规则备份失败，未修改"
		return s
	}
	sum := sha256.Sum256(before)
	if e = record(dir, Change{Kind: "verge-rules", File: file, Backup: backup, BeforeHash: hex.EncodeToString(sum[:])}); e != nil {
		s.Status = "error"
		s.Detail = "无法保存撤销记录，未修改"
		return s
	}
	if e = replaceChecked(file, before, after); e != nil {
		s.Status = "changed-config"
		s.Detail = "规则文件发生变化或无法写入，未覆盖；请在客户端处理"
	}
	return s
}
func vergeRulePath(root string) (string, error) {
	b, e := readSafe(filepath.Join(root, "profiles.yaml"))
	if e != nil {
		return "", e
	}
	n, e := doc(b)
	if e != nil {
		return "", e
	}
	current := scalar(n, "current")
	items := field(n, "items")
	if current == "" || items == nil || items.Kind != yaml.SequenceNode {
		return "", errors.New("no current profile")
	}
	byID := map[string]*yaml.Node{}
	for _, item := range items.Content {
		id := scalar(item, "uid")
		if id != "" {
			if byID[id] != nil {
				return "", errors.New("duplicate profile UID")
			}
			byID[id] = item
		}
	}
	selected := byID[current]
	if selected == nil {
		return "", errors.New("missing selected profile")
	}
	ruleID := scalar(field(selected, "option"), "rules")
	item := byID[ruleID]
	if item == nil || scalar(item, "type") != "rules" {
		return "", errors.New("missing rule extension")
	}
	file := scalar(item, "file")
	if file == "" || filepath.Base(file) != file || strings.ContainsAny(file, "/\\:") || !strings.HasSuffix(file, ".yaml") {
		return "", errors.New("unsafe extension path")
	}
	target := filepath.Join(root, "profiles", file)
	if _, e = readSafe(target); e != nil {
		return "", e
	}
	return target, nil
}
func vergeRoots() []string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return []string{filepath.Join(home, "Library", "Application Support", "io.github.clash-verge-rev.clash-verge-rev")}
	}
	if runtime.GOOS == "windows" {
		return []string{filepath.Join(os.Getenv("APPDATA"), "io.github.clash-verge-rev.clash-verge-rev")}
	}
	return nil
}
func configure(dir string, apply bool, run Runner) []Step {
	names := runningClients(run)
	out := []Step{{Name: "环境", Status: "ok", Detail: runtime.GOOS + "/" + runtime.GOARCH + "；无需 Node 或 Git"}}
	if len(names) > 0 {
		out = append(out, Step{Name: "代理客户端", Status: "detected", Detail: strings.Join(names, ", ")})
	}
	hasVerge := false
	for _, n := range names {
		if n == "Clash Verge" {
			hasVerge = true
		}
	}
	if hasVerge {
		var files []string
		for _, root := range vergeRoots() {
			if f, e := vergeRulePath(root); e == nil {
				files = append(files, f)
			}
		}
		if len(files) == 1 {
			return append(out, applyYAML(files[0], dir, apply))
		}
		out = append(out, Step{Name: "客户端规则", Status: "needs-agent", Detail: "Clash Verge 当前规则文件不唯一或不存在。请在当前订阅的编辑规则中前置添加 " + directRule})
	}
	out = append(out, systemBypass(dir, apply, run)...)
	if len(names) > 0 && !hasVerge {
		out = append(out, Step{Name: "客户端路由", Status: "needs-agent", Detail: "检测到 " + strings.Join(names, ", ") + "；请确认本站在 Rule/TUN 模式也命中 DIRECT。系统代理例外不能控制 VPN/TUN。"})
	}
	return out
}
func undo(dir string, run Runner) []Step {
	s, e := loadState(dir)
	if e != nil {
		return []Step{{Name: "撤销", Status: "error", Detail: "无法读取本机撤销记录"}}
	}
	out := []Step{}
	remaining := []Change{}
	for _, c := range s.Changes {
		step := Step{Name: "撤销本站直连", Status: "restored", File: c.File}
		if c.Kind == "verge-rules" {
			b, e := readSafe(c.File)
			if e == nil {
				var after []byte
				var changed bool
				after, changed, e = editYAML(b, true)
				if e == nil && changed {
					e = replaceChecked(c.File, b, after)
				}
			}
			if e != nil {
				step.Status = "changed-config"
				step.Detail = "规则已更改，未覆盖；请手动移除 INKWARD 标记的本站规则"
				remaining = append(remaining, c)
			} else {
				step.Status = "needs-reload"
				step.Detail = "本站规则已撤销，请重载 Clash Verge 当前配置"
			}
		} else {
			var err error
			step, err = undoSystem(c, run)
			if err != nil {
				remaining = append(remaining, c)
			}
		}
		out = append(out, step)
	}
	s.Changes = remaining
	b, _ := json.MarshalIndent(s, "", "  ")
	if len(out) > 0 {
		if e = atomicWrite(filepath.Join(dir, "state.json"), b, 0600); e != nil {
			out = append(out, Step{Name: "撤销记录", Status: "error", Detail: "无法更新撤销记录"})
		}
	} else {
		out = append(out, Step{Name: "撤销", Status: "no-changes", Detail: "没有本站修改记录"})
	}
	return out
}
func mergeDomains(before []string) ([]string, []string) {
	after := append([]string{}, before...)
	added := []string{}
	for _, target := range []string{"rimbaud.asia", "*.rimbaud.asia"} {
		found := false
		for _, x := range after {
			if strings.EqualFold(strings.TrimSpace(x), target) {
				found = true
			}
		}
		if !found {
			after = append(after, target)
			added = append(added, target)
		}
	}
	return after, added
}
func removeDomains(current, added []string) []string {
	out := []string{}
	for _, x := range current {
		remove := false
		for _, a := range added {
			if (a == "rimbaud.asia" || a == "*.rimbaud.asia") && strings.EqualFold(x, a) {
				remove = true
			}
		}
		if !remove {
			out = append(out, x)
		}
	}
	return out
}

func lockState(dir string) (func(), error) {
	if e := ensureDir(dir); e != nil {
		return nil, e
	}
	p := filepath.Join(dir, "setup.lock")
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, e
	}
	fmt.Fprintln(f, os.Getpid())
	f.Close()
	return func() { _ = os.Remove(p) }, nil
}

func hasAlias(n *yaml.Node) bool {
	if n.Alias != nil || n.Anchor != "" || n.Kind == yaml.AliasNode {
		return true
	}
	for _, c := range n.Content {
		if hasAlias(c) {
			return true
		}
	}
	return false
}
