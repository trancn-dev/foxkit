package proj

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PHPInfo struct {
	Require   string            `json:"require"`
	Framework string            `json:"framework,omitempty"`
	Version   string            `json:"version,omitempty"`
	Packages  map[string]string `json:"packages"`
	Installed bool              `json:"installed"`
}

type NodeInfo struct {
	PackageManager string `json:"packageManager"`
	Engines        string `json:"engines,omitempty"`
	Nvmrc          string `json:"nvmrc,omitempty"`
	Framework      string `json:"framework,omitempty"`
	FrameworkVer   string `json:"frameworkVersion,omitempty"`
	Bundler        string `json:"bundler,omitempty"`
	Installed      bool   `json:"installed"`
}

type GoInfo struct {
	Module    string `json:"module"`
	GoVersion string `json:"goVersion"`
}

type GitInfo struct {
	Branch string `json:"branch"`
}

type Script struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Run     string `json:"run"`
}

type Info struct {
	Dir      string    `json:"dir"`
	Name     string    `json:"name"`
	Types    []string  `json:"types"`
	PHP      *PHPInfo  `json:"php,omitempty"`
	Node     *NodeInfo `json:"node,omitempty"`
	Go       *GoInfo   `json:"go,omitempty"`
	Git      *GitInfo  `json:"git,omitempty"`
	Scripts  []Script  `json:"scripts"`
	Warnings []string  `json:"warnings"`
}

// FileError: file cấu hình hỏng (composer.json / package.json).
type FileError struct {
	File string
	Err  error
}

func (e *FileError) Error() string { return fmt.Sprintf("%s hỏng: %v", e.File, e.Err) }
func (e *FileError) Unwrap() error { return e.Err }

// notable: gói đáng chú ý, danh sách cố định.
var notable = []string{"laravel/sanctum", "laravel/passport", "laravel/horizon", "laravel/octane", "livewire/livewire", "inertiajs/inertia-laravel", "filament/filament", "spatie/laravel-permission"}

func readJSON(path string, v any) (bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return true, &FileError{filepath.Base(path), err}
	}
	if err := json.Unmarshal(b, v); err != nil {
		return true, &FileError{filepath.Base(path), err}
	}
	return true, nil
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func add(list *[]string, s string) {
	for _, x := range *list {
		if x == s {
			return
		}
	}
	*list = append(*list, s)
}

// Detect nhìn vào một thư mục và cho biết loại dự án, phiên bản thật đang cài và các script chạy được.
func Detect(dir string) (Info, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Info{}, err
	}
	in := Info{Dir: abs, Name: filepath.Base(abs), Types: []string{}, Scripts: []Script{}, Warnings: []string{}}
	if err := detectComposer(&in); err != nil {
		return in, err
	}
	if err := detectNode(&in); err != nil {
		return in, err
	}
	detectGo(&in)
	detectGit(&in)
	return in, nil
}

type composerJSON struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
	Scripts    map[string]any    `json:"scripts"`
}

type composerLock struct {
	Packages    []lockPkg `json:"packages"`
	PackagesDev []lockPkg `json:"packages-dev"`
}

type lockPkg struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func detectComposer(in *Info) error {
	var c composerJSON
	found, err := readJSON(filepath.Join(in.Dir, "composer.json"), &c)
	if err != nil || !found {
		return err
	}
	php := &PHPInfo{Require: c.Require["php"], Packages: map[string]string{}, Installed: exists(filepath.Join(in.Dir, "vendor"))}
	in.PHP = php
	versions := map[string]string{}
	var lock composerLock
	hasLock, err := readJSON(filepath.Join(in.Dir, "composer.lock"), &lock)
	if err != nil {
		return err
	}
	for _, p := range append(lock.Packages, lock.PackagesDev...) {
		versions[p.Name] = strings.TrimPrefix(p.Version, "v")
	}
	if !hasLock {
		in.Warnings = append(in.Warnings, "chưa có composer.lock: phiên bản chỉ là ràng buộc trong composer.json")
	}
	ver := func(name string) string {
		if v, ok := versions[name]; ok {
			return v
		}
		if v, ok := c.Require[name]; ok {
			return v
		}
		return c.RequireDev[name]
	}
	for _, n := range notable {
		if _, ok := c.Require[n]; ok || c.RequireDev[n] != "" {
			php.Packages[strings.TrimPrefix(strings.TrimPrefix(n, "laravel/"), "spatie/")] = ver(n)
		}
	}
	isLaravel := c.Require["laravel/framework"] != "" && exists(filepath.Join(in.Dir, "artisan"))
	if isLaravel {
		php.Framework, php.Version = "laravel", ver("laravel/framework")
		add(&in.Types, "laravel")
	} else {
		add(&in.Types, "php")
	}
	names := make([]string, 0, len(c.Scripts))
	for n := range c.Scripts {
		if strings.HasPrefix(n, "pre-") || strings.HasPrefix(n, "post-") {
			continue // hook tự động, không phải lệnh để chạy tay
		}
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		in.Scripts = append(in.Scripts, Script{Source: "composer", Name: n, Command: scriptCommand(c.Scripts[n]), Run: "composer " + n})
	}
	if isLaravel {
		for _, a := range []string{"serve", "migrate", "queue:work", "test"} {
			cmd := "php artisan " + a
			in.Scripts = append(in.Scripts, Script{Source: "artisan", Name: a, Command: cmd, Run: cmd})
		}
		if exists(filepath.Join(in.Dir, ".env.example")) && !exists(filepath.Join(in.Dir, ".env")) {
			in.Warnings = append(in.Warnings, "chưa có .env: copy .env.example .env")
		}
	}
	if !php.Installed {
		in.Warnings = append(in.Warnings, "chưa cài: chạy composer install")
	}
	return nil
}

func scriptCommand(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		parts := make([]string, 0, len(x))
		for _, p := range x {
			parts = append(parts, fmt.Sprint(p))
		}
		return strings.Join(parts, " && ")
	}
	return ""
}
