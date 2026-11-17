# Spec: `foxkit proj`

> Nhóm: **dev** · Giai đoạn: GĐ5 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Nhìn vào một thư mục và cho biết: đây là dự án gì (Laravel, Vue 2/3, Nuxt, Vite, Node, Go...), phiên bản framework **thật đang cài** (từ lock file), yêu cầu PHP/Node, và danh sách script chạy được. Launcher dùng `scripts` để sinh button động ("npm run dev", "composer test") cho dự án đang chọn.

## 2. Cú pháp
```
foxkit proj [dir] [--json]
```

## 3. File
```
cmd/proj.go
internal/proj/detect.go
internal/proj/composer.go
internal/proj/node.go
internal/proj/golang.go
internal/proj/proj_test.go
internal/proj/testdata/{laravel11-vue2, vue3-vite, nuxt3, go-cli}/...
```

## 4. Hợp đồng
```go
package proj
type Info struct {
    Dir      string        `json:"dir"`
    Name     string        `json:"name"`
    Types    []string      `json:"types"`             // ["laravel","vue"]
    PHP      *PHPInfo      `json:"php,omitempty"`
    Node     *NodeInfo     `json:"node,omitempty"`
    Go       *GoInfo       `json:"go,omitempty"`
    Git      *GitInfo      `json:"git,omitempty"`
    Scripts  []Script      `json:"scripts"`
    Warnings []string      `json:"warnings"`
}
type PHPInfo struct {
    Require   string            `json:"require"`            // "^8.2"
    Framework string            `json:"framework,omitempty"`// "laravel"
    Version   string            `json:"version,omitempty"`  // "11.31.0" (từ composer.lock)
    Packages  map[string]string `json:"packages"`           // gói đáng chú ý: livewire, inertia, filament, sanctum, horizon...
    Installed bool              `json:"installed"`          // có vendor/
}
type NodeInfo struct {
    PackageManager string            `json:"packageManager"` // npm|yarn|pnpm|bun (theo lock file)
    Engines        string            `json:"engines,omitempty"`
    Nvmrc          string            `json:"nvmrc,omitempty"`
    Framework      string            `json:"framework,omitempty"` // vue|nuxt|react|next|svelte
    FrameworkVer   string            `json:"frameworkVersion,omitempty"`
    Bundler        string            `json:"bundler,omitempty"`    // vite|laravel-mix|webpack|vue-cli
    Installed      bool              `json:"installed"`            // có node_modules/
}
type GoInfo struct { Module string `json:"module"`; GoVersion string `json:"goVersion"` }
type GitInfo struct { Branch string `json:"branch"` }
type Script struct {
    Source  string `json:"source"`   // npm|composer|artisan
    Name    string `json:"name"`     // "dev"
    Command string `json:"command"`  // lệnh gốc trong file: "vite"
    Run     string `json:"run"`      // lệnh để chạy: "npm run dev"
}
func Detect(dir string) (Info, error)
```

## 5. Cách hoạt động
1. **composer.json** có → đọc `require.php`, các gói trong `require`/`require-dev`. Có `laravel/framework` + file `artisan` → `laravel`.
   - Version thật: đọc `composer.lock` → mảng `packages[].name/version` (bỏ tiền tố `v`). Không có lock → dùng ràng buộc trong composer.json, kèm warning "chưa có composer.lock".
   - Gói đáng chú ý (danh sách cố định): `laravel/sanctum`, `laravel/passport`, `laravel/horizon`, `laravel/octane`, `livewire/livewire`, `inertiajs/inertia-laravel`, `filament/filament`, `spatie/laravel-permission`.
   - Scripts: `composer.json` `scripts` (bỏ hook tự động như `post-autoload-dump`, `pre-*`, `post-*`) → `composer <name>`. Laravel thêm sẵn vài script artisan hay dùng: `php artisan serve`, `php artisan migrate`, `php artisan queue:work`, `php artisan test` (Source `artisan`).
2. **package.json** có → package manager theo lock: `pnpm-lock.yaml` → pnpm, `yarn.lock` → yarn, `bun.lockb`/`bun.lock` → bun, còn lại npm. Trường `packageManager` (Corepack) được ưu tiên nếu có.
   - Framework/version: kiểm `dependencies` + `devDependencies`: `nuxt` → nuxt, `vue` → vue (Vue 2 hay 3 theo version thật), `next`, `react`, `svelte`. Version thật lấy từ `node_modules/<pkg>/package.json` nếu đã cài, không thì từ `package-lock.json` (`packages["node_modules/vue"].version`), cuối cùng dùng ràng buộc.
   - Bundler: `vite` → vite; `laravel-mix` → laravel-mix; `@vue/cli-service` → vue-cli; `webpack` → webpack.
   - `engines.node`, `.nvmrc`/`.node-version`.
   - Scripts: `package.json scripts` → `<pm> run <name>` (yarn/bun: `yarn <name>`).
3. **go.mod** → module, go version.
4. **.git/HEAD** → branch (`ref: refs/heads/<branch>`; detached → 7 ký tự hash).
5. Warnings:
   - Có package.json mà thiếu node_modules → "chưa cài: chạy <pm> install".
   - Có composer.json mà thiếu vendor → "chưa cài: chạy composer install".
   - Laravel có `.env.example` mà thiếu `.env` → "chưa có .env: copy .env.example .env".
   - Nhiều lock file JS cùng lúc → cảnh báo xung đột.
   - Vue 2 → "Vue 2 đã hết hỗ trợ (EOL 31/12/2023)".
6. Không nhận diện được gì → `types: []`, exit 0 với thông báo.

## 6. Output
```
shop-api  (D:\Projects\shop-api)  · branch feature/coupon
Loại     Laravel 11.31.0 + Vue 2.7.16 (Vite)
PHP      ^8.2  · sanctum 4.0.3, horizon 5.29.1
Node     npm · engines >=18 · .nvmrc 20

Scripts
  npm run dev          vite
  npm run build        vite build
  composer test        php artisan test
  php artisan serve
  ...
Cảnh báo
  ! chưa có .env: copy .env.example .env
  ! Vue 2 đã hết hỗ trợ (EOL 31/12/2023)
```
JSON: `{ "ok": true, "data": <Info> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIR_NOT_FOUND` | | 2 | |
| `PROJ_FILE_INVALID` | composer.json/package.json hỏng | 1 | Kiểm tra bằng foxkit json <file> --validate |

## 8. Manifest
```json
{ "name": "proj", "title": "Thông tin dự án", "description": "Nhận diện Laravel/Vue/Node/Go, phiên bản, scripts", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Thư mục dự án" } ],
  "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
Fixture `testdata/` (chỉ file json/lock tối thiểu, không có code):
- `laravel11-vue2`: composer.json + composer.lock (laravel 11.31.0) + package.json (vue ^2.7, vite) + package-lock.json (vue 2.7.16) + artisan rỗng → types `[laravel, vue]`, đúng version, warning thiếu `.env`.
- `vue3-vite` với `pnpm-lock.yaml` → pm pnpm, `pnpm run dev`.
- `nuxt3` → framework nuxt (không báo thêm vue).
- `go-cli` → go module.
- Thư mục rỗng → types rỗng.

Chạy thử: `.\foxkit proj D:\Projects\shop-api`, `.\foxkit proj --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + fixture test.

## 11. Ngoài phạm vi
Chạy script (launcher làm), kiểm tra phiên bản PHP/Node đang cài có khớp yêu cầu không (có thể thêm sau bằng `which --version`), monorepo nhiều package.

## 12. Go học được
Decode JSON vào struct có tag và field tùy chọn, `map[string]string` cho phần động, con trỏ struct để `omitempty` bỏ cả khối, fixture `testdata/` mô phỏng dự án thật.
