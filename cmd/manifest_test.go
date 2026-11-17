package cmd

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"foxkit/internal/manifest"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func runnable(c *cobra.Command, out map[string]*cobra.Command) {
	for _, sub := range c.Commands() {
		if sub.Hidden || sub.Name() == "help" {
			continue
		}
		if sub.Runnable() {
			out[strings.TrimPrefix(sub.CommandPath(), "foxkit ")] = sub
		}
		runnable(sub, out)
	}
}

func in(s string, list ...string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestManifestMatchesCobra(t *testing.T) {
	f, err := manifest.Parse()
	if err != nil {
		t.Fatalf("commands.json hỏng: %v", err)
	}
	if f.Schema != 1 {
		t.Fatalf("schema = %d, cần 1", f.Schema)
	}
	cobraCmds := map[string]*cobra.Command{}
	runnable(rootCmd, cobraCmds)
	entries := map[string]manifest.Command{}
	for _, e := range f.Commands {
		if _, dup := entries[e.Name]; dup {
			t.Errorf("entry %q bị trùng trong internal/manifest/commands.json", e.Name)
		}
		entries[e.Name] = e
	}
	for name := range cobraCmds {
		if _, ok := entries[name]; !ok {
			t.Errorf("lệnh %q có trong cobra nhưng thiếu trong internal/manifest/commands.json", name)
		}
	}
	for name, e := range entries {
		cc, ok := cobraCmds[name]
		if !ok {
			t.Errorf("entry %q có trong internal/manifest/commands.json nhưng không có lệnh cobra tương ứng", name)
			continue
		}
		var have []string
		cc.LocalFlags().VisitAll(func(fl *pflag.Flag) {
			if fl.Name != "help" {
				have = append(have, fl.Name)
			}
		})
		var want []string
		for _, fl := range e.Flags {
			want = append(want, fl.Name)
			if !in(fl.Type, "string", "text", "int", "bool", "enum", "file", "dir", "list") {
				t.Errorf("lệnh %q flag %q: type %q không hợp lệ", name, fl.Name, fl.Type)
			}
		}
		for _, a := range e.Args {
			if !in(a.Type, "string", "text", "int", "bool", "enum", "file", "dir", "list") {
				t.Errorf("lệnh %q arg %q: type %q không hợp lệ", name, a.Name, a.Type)
			}
		}
		sort.Strings(have)
		sort.Strings(want)
		if strings.Join(have, ",") != strings.Join(want, ",") {
			t.Errorf("lệnh %q: cờ cobra [%s] khác flags trong manifest [%s]", name, strings.Join(have, ","), strings.Join(want, ","))
		}
		if !in(e.Category, "core", "dev", "life", "system", "network") {
			t.Errorf("lệnh %q: category %q không hợp lệ", name, e.Category)
		}
		if !in(e.Output, "kv", "table", "text", "stream") {
			t.Errorf("lệnh %q: output %q không hợp lệ", name, e.Output)
		}
		if len(e.Examples) > 3 {
			t.Errorf("lệnh %q: tối đa 3 examples", name)
		}
		for _, ex := range e.Examples {
			if ex.Cmd != name && !strings.HasPrefix(ex.Cmd, name+" ") {
				t.Errorf("lệnh %q: example %q phải bắt đầu bằng tên lệnh", name, ex.Cmd)
			}
			if strings.Contains(ex.Cmd, "--json") {
				t.Errorf("lệnh %q: example %q không được chứa --json", name, ex.Cmd)
			}
		}
	}
}

func execute(args ...string) (stdout, stderr string, code int) {
	var o, e bytes.Buffer
	rootCmd.SetOut(&o)
	rootCmd.SetErr(&e)
	defer func() { rootCmd.SetOut(nil); rootCmd.SetErr(nil); resetFlags(rootCmd) }()
	code = run(args)
	return o.String(), e.String(), code
}

// resetFlags đưa cờ về mặc định giữa các lần chạy trong cùng process test.
func resetFlags(c *cobra.Command) {
	reset := func(fl *pflag.Flag) {
		if fl.Changed {
			_ = fl.Value.Set(fl.DefValue) // DefValue luôn hợp lệ với chính cờ đó
			fl.Changed = false
		}
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, s := range c.Commands() {
		resetFlags(s)
	}
}

func TestVersionAndErrors(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		inOut    string
		inErr    string
	}{
		{"version text", []string{"version"}, 0, "foxkit ", ""},
		{"version json", []string{"version", "--json"}, 0, `"ok": true`, ""},
		{"manifest json", []string{"manifest", "--json"}, 0, `"schema": 1`, ""},
		{"unknown cmd", []string{"khongco"}, 2, "", "Lỗi:"},
		{"unknown cmd json", []string{"khongco", "--json"}, 2, "", `"ok": false`},
		{"bad flag", []string{"version", "--zzz"}, 2, "", "Lỗi:"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, e, code := execute(tc.args...)
			if code != tc.wantExit || !strings.Contains(o, tc.inOut) || !strings.Contains(e, tc.inErr) {
				t.Fatalf("exit %d out %q err %q", code, o, e)
			}
		})
	}
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	o, _, _ := execute("version", "--json")
	if err := json.Unmarshal([]byte(o), &env); err != nil || !env.OK {
		t.Fatalf("envelope sai: %q %v", o, err)
	}
}
