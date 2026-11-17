package proj

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture dựng cây file tối thiểu (chỉ json/lock, không có code) trong thư mục tạm.
func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	d := t.TempDir()
	for name, body := range files {
		p := filepath.Join(d, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestLaravelWithVue2(t *testing.T) {
	d := fixture(t, map[string]string{
		"artisan":           "",
		".env.example":      "APP_NAME=x",
		".git/HEAD":         "ref: refs/heads/feature/coupon\n",
		"composer.json":     `{"require":{"php":"^8.2","laravel/framework":"^11.0","laravel/sanctum":"^4.0"},"scripts":{"test":"php artisan test","post-autoload-dump":"x","dev":["a","b"]}}`,
		"composer.lock":     `{"packages":[{"name":"laravel/framework","version":"v11.31.0"},{"name":"laravel/sanctum","version":"v4.0.3"}]}`,
		"package.json":      `{"dependencies":{"vue":"^2.7.0"},"devDependencies":{"vite":"^5"},"scripts":{"dev":"vite","build":"vite build"},"engines":{"node":">=18"}}`,
		"package-lock.json": `{"packages":{"node_modules/vue":{"version":"2.7.16"}}}`,
		".nvmrc":            "20\n",
	})
	in, err := Detect(d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(in.Types, ",") != "laravel,vue" {
		t.Fatalf("types %v", in.Types)
	}
	if in.PHP.Version != "11.31.0" || in.PHP.Require != "^8.2" || in.PHP.Packages["sanctum"] != "4.0.3" {
		t.Errorf("php: %+v", in.PHP)
	}
	if in.Node.FrameworkVer != "2.7.16" || in.Node.Bundler != "vite" || in.Node.PackageManager != "npm" || in.Node.Nvmrc != "20" || in.Node.Engines != ">=18" {
		t.Errorf("node: %+v", in.Node)
	}
	if in.Git == nil || in.Git.Branch != "feature/coupon" {
		t.Errorf("git: %+v", in.Git)
	}
	runs := map[string]string{}
	for _, s := range in.Scripts {
		runs[s.Run] = s.Command
	}
	for _, want := range []string{"npm run dev", "npm run build", "composer test", "composer dev", "php artisan serve", "php artisan test"} {
		if _, ok := runs[want]; !ok {
			t.Errorf("thiếu script %q (có %v)", want, runs)
		}
	}
	if _, bad := runs["composer post-autoload-dump"]; bad {
		t.Error("hook post-* không phải script")
	}
	w := strings.Join(in.Warnings, "|")
	for _, s := range []string{"chưa có .env", "Vue 2 đã hết hỗ trợ", "composer install", "npm install"} {
		if !strings.Contains(w, s) {
			t.Errorf("thiếu cảnh báo %q trong %q", s, w)
		}
	}
}

func TestVue3PnpmAndNuxt(t *testing.T) {
	d := fixture(t, map[string]string{
		"package.json":                  `{"dependencies":{"vue":"^3.4.0"},"devDependencies":{"vite":"^5"},"scripts":{"dev":"vite"}}`,
		"pnpm-lock.yaml":                "lockfileVersion: 9",
		"node_modules/vue/package.json": `{"version":"3.5.1"}`,
	})
	in, _ := Detect(d)
	if in.Node.PackageManager != "pnpm" || in.Scripts[0].Run != "pnpm run dev" || in.Node.FrameworkVer != "3.5.1" || !in.Node.Installed {
		t.Fatalf("%+v %+v", in.Node, in.Scripts)
	}
	if has(in.Warnings, "chưa cài: chạy pnpm install") {
		t.Error("đã có node_modules thì không cảnh báo")
	}
	n := fixture(t, map[string]string{"package.json": `{"dependencies":{"nuxt":"^3.0.0","vue":"^3.4.0"},"packageManager":"yarn@4.0.0"}`})
	in, _ = Detect(n)
	if strings.Join(in.Types, ",") != "nuxt" || in.Node.PackageManager != "yarn" {
		t.Fatalf("nuxt không được báo thêm vue; packageManager ưu tiên: %v %+v", in.Types, in.Node)
	}
}

func TestGoAndEmptyAndConflictAndBroken(t *testing.T) {
	in, _ := Detect(fixture(t, map[string]string{"go.mod": "module foxkit\n\ngo 1.22\n"}))
	if in.Go == nil || in.Go.Module != "foxkit" || in.Go.GoVersion != "1.22" || strings.Join(in.Types, ",") != "go" {
		t.Fatalf("%+v", in)
	}
	in, err := Detect(t.TempDir())
	if err != nil || in.Types == nil || len(in.Types) != 0 || in.Scripts == nil || in.Warnings == nil {
		t.Fatalf("thư mục rỗng: %+v %v", in, err)
	}
	in, _ = Detect(fixture(t, map[string]string{"package.json": `{}`, "yarn.lock": "", "package-lock.json": "{}"}))
	if !strings.Contains(strings.Join(in.Warnings, "|"), "nhiều lock file") {
		t.Errorf("phải cảnh báo xung đột lock: %v", in.Warnings)
	}
	_, err = Detect(fixture(t, map[string]string{"composer.json": "{ hỏng"}))
	var fe *FileError
	if !errors.As(err, &fe) || fe.File != "composer.json" {
		t.Fatalf("%v", err)
	}
}

func TestVendorOfGoNotComposer(t *testing.T) {
	in, _ := Detect(fixture(t, map[string]string{"go.mod": "module x\n", "vendor/modules.txt": ""}))
	if in.PHP != nil {
		t.Error("không có composer.json thì không có PHP")
	}
}
