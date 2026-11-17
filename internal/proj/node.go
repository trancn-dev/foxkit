package proj

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
	Engines         map[string]string `json:"engines"`
	PackageManager  string            `json:"packageManager"`
}

type npmLock struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
}

var lockFiles = []struct{ file, pm string }{{"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"bun.lockb", "bun"}, {"bun.lock", "bun"}, {"package-lock.json", "npm"}}

func runCmd(pm, script string) string {
	switch pm {
	case "yarn":
		return "yarn " + script
	case "npm", "pnpm", "bun":
		return pm + " run " + script
	}
	return "npm run " + script
}

func detectNode(in *Info) error {
	var p packageJSON
	found, err := readJSON(filepath.Join(in.Dir, "package.json"), &p)
	if err != nil || !found {
		return err
	}
	n := &NodeInfo{PackageManager: "npm", Installed: exists(filepath.Join(in.Dir, "node_modules")), Engines: p.Engines["node"]}
	in.Node = n

	// package manager: trường packageManager (Corepack) > lock file > npm
	var locks []string
	for _, l := range lockFiles {
		if exists(filepath.Join(in.Dir, l.file)) {
			locks = append(locks, l.file)
			if len(locks) == 1 {
				n.PackageManager = l.pm
			}
		}
	}
	if pm, _, ok := strings.Cut(p.PackageManager, "@"); ok && pm != "" {
		n.PackageManager = pm
	}
	if len(locks) > 1 {
		in.Warnings = append(in.Warnings, "nhiều lock file JS cùng lúc ("+strings.Join(locks, ", ")+"): dễ xung đột phiên bản")
	}
	for _, f := range []string{".nvmrc", ".node-version"} {
		if b, err := os.ReadFile(filepath.Join(in.Dir, f)); err == nil {
			n.Nvmrc = strings.TrimSpace(string(b))
			break
		}
	}

	deps := map[string]string{}
	for k, v := range p.Dependencies {
		deps[k] = v
	}
	for k, v := range p.DevDependencies {
		deps[k] = v
	}
	realVersion := func(pkg string) string {
		var nm struct {
			Version string `json:"version"`
		}
		if ok, _ := readJSON(filepath.Join(in.Dir, "node_modules", filepath.FromSlash(pkg), "package.json"), &nm); ok && nm.Version != "" {
			return nm.Version
		}
		var lock npmLock
		if ok, _ := readJSON(filepath.Join(in.Dir, "package-lock.json"), &lock); ok {
			if v := lock.Packages["node_modules/"+pkg].Version; v != "" {
				return v
			}
		}
		return strings.TrimLeft(deps[pkg], "^~>=< ")
	}
	// nuxt/next đã bao gồm vue/react nên không báo thêm
	for _, fw := range []string{"nuxt", "next", "vue", "react", "svelte"} {
		if _, ok := deps[fw]; ok {
			n.Framework, n.FrameworkVer = fw, realVersion(fw)
			break
		}
	}
	if n.Framework != "" {
		add(&in.Types, n.Framework)
	} else {
		add(&in.Types, "node")
	}
	if n.Framework == "vue" && strings.HasPrefix(n.FrameworkVer, "2") {
		in.Warnings = append(in.Warnings, "Vue 2 đã hết hỗ trợ (EOL 31/12/2023)")
	}
	for _, b := range []struct{ pkg, name string }{{"vite", "vite"}, {"laravel-mix", "laravel-mix"}, {"@vue/cli-service", "vue-cli"}, {"webpack", "webpack"}} {
		if _, ok := deps[b.pkg]; ok {
			n.Bundler = b.name
			break
		}
	}
	names := make([]string, 0, len(p.Scripts))
	for k := range p.Scripts {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		in.Scripts = append(in.Scripts, Script{Source: "npm", Name: k, Command: p.Scripts[k], Run: runCmd(n.PackageManager, k)})
	}
	if !n.Installed {
		in.Warnings = append(in.Warnings, "chưa cài: chạy "+n.PackageManager+" install")
	}
	return nil
}

func detectGo(in *Info) {
	f, err := os.Open(filepath.Join(in.Dir, "go.mod"))
	if err != nil {
		return
	}
	defer f.Close()
	g := &GoInfo{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if v, ok := strings.CutPrefix(line, "module "); ok {
			g.Module = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(line, "go "); ok && g.GoVersion == "" {
			g.GoVersion = strings.TrimSpace(v)
		}
	}
	in.Go = g
	add(&in.Types, "go")
}

func detectGit(in *Info) {
	b, err := os.ReadFile(filepath.Join(in.Dir, ".git", "HEAD"))
	if err != nil {
		return
	}
	head := strings.TrimSpace(string(b))
	if ref, ok := strings.CutPrefix(head, "ref: refs/heads/"); ok {
		in.Git = &GitInfo{Branch: ref}
	} else if len(head) >= 7 {
		in.Git = &GitInfo{Branch: head[:7]} // detached: 7 ký tự hash
	}
}
