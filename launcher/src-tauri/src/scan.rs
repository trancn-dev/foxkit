//! Quét lệnh ngoài (launcher.md 6.9): đọc `foxkit.json` ở thư mục gốc và thư mục con trực tiếp, lưu snapshot.
//! Không quét tự động; chỉ gọi khi người dùng `:add` / `:scan`.
use serde::{Deserialize, Serialize};
use serde_json::{json, Map, Value};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, HashMap, HashSet};
use std::path::{Path, PathBuf};

const MAX_FILE_BYTES: u64 = 256 * 1024;
const MAX_COMMANDS: usize = 200;
const SKIP_DIRS: [&str; 7] = ["node_modules", ".git", "target", "vendor", "dist", "bin", "obj"];
const PARAM_TYPES: [&str; 8] = ["string", "text", "int", "bool", "enum", "file", "dir", "list"];

#[derive(Serialize, Deserialize, Clone, Debug, Default)]
pub struct FileSnap {
    pub path: String,
    pub sha256: String,
    pub commands: Vec<Value>,
}

#[derive(Serialize, Deserialize, Clone, Debug, Default)]
pub struct FolderSnap {
    pub path: String,
    pub scanned_at: u64,
    /// "ok" | "missing"
    pub status: String,
    pub files: Vec<FileSnap>,
    /// Số file foxkit.json bị bỏ ở lần quét gần nhất.
    #[serde(default)]
    pub skipped: u32,
}

#[derive(Serialize, Deserialize, Clone, Debug, Default)]
pub struct Snapshot {
    pub folders: Vec<FolderSnap>,
}

impl Snapshot {
    fn commands(&self) -> HashMap<String, &Value> {
        let mut m = HashMap::new();
        for c in self.folders.iter().flat_map(|f| &f.files).flat_map(|f| &f.commands) {
            if let Some(n) = c.get("name").and_then(Value::as_str) {
                m.insert(n.to_string(), c);
            }
        }
        m
    }

    /// Mọi lệnh ngoài, theo thứ tự thư mục → file → lệnh.
    pub fn all_commands(&self) -> Vec<Value> {
        self.folders.iter().flat_map(|f| &f.files).flat_map(|f| f.commands.iter().cloned()).collect()
    }
}

#[derive(Serialize, Debug, Default, PartialEq)]
pub struct ScanReport {
    pub added: u32,
    pub changed: u32,
    pub removed: u32,
    pub skipped_files: u32,
    pub folders_missing: Vec<String>,
    /// Thông điệp hiện cho người dùng (vd "Không thấy foxkit.json nào trong ...").
    pub messages: Vec<String>,
}

pub fn sha256_hex(bytes: &[u8]) -> String {
    Sha256::digest(bytes).iter().map(|b| format!("{b:02x}")).collect()
}

fn valid_name(n: &str) -> bool {
    !n.is_empty()
        && n.split(' ').all(|p| {
            let mut ch = p.chars();
            ch.next().is_some_and(|c| c.is_ascii_lowercase() || c.is_ascii_digit()) && ch.all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '-')
        })
}

/// Chỉ .exe/.bat/.cmd/.com hoặc không có đuôi (tên trong PATH, vd `powershell`); `.ps1`... phải chạy qua trình thông dịch.
fn program_ok(p: &str) -> bool {
    let last = p.rsplit(['\\', '/']).next().unwrap_or(p);
    match Path::new(last).extension().and_then(|e| e.to_str()) {
        None => true,
        Some(e) => matches!(e.to_ascii_lowercase().as_str(), "exe" | "bat" | "cmd" | "com"),
    }
}

fn check_params(o: &Map<String, Value>, key: &str) -> Result<(), String> {
    let Some(v) = o.get(key) else { return Ok(()) };
    let arr = v.as_array().ok_or(format!("{key} phải là mảng"))?;
    for p in arr {
        let name = p.get("name").and_then(Value::as_str).filter(|s| !s.is_empty()).ok_or(format!("{key}: thiếu name"))?;
        let ty = p.get("type").and_then(Value::as_str).unwrap_or("");
        if !PARAM_TYPES.contains(&ty) {
            return Err(format!("{key} \"{name}\": type \"{ty}\" không hợp lệ"));
        }
    }
    Ok(())
}

/// Kiểm một lệnh và chuẩn hóa (điền mặc định, thêm source/file/baseDir).
fn parse_command(v: &Value, file: &str, base: &str) -> Result<Value, String> {
    let o = v.as_object().ok_or("lệnh không phải object")?;
    let name = o.get("name").and_then(Value::as_str).ok_or("thiếu name")?;
    if !valid_name(name) {
        return Err(format!("name \"{name}\" sai quy tắc (chữ thường, số, -, dấu cách cho lệnh con)"));
    }
    let exec = o.get("exec").and_then(Value::as_object).ok_or("thiếu khối exec")?;
    let program = exec.get("program").and_then(Value::as_str).filter(|s| !s.is_empty()).ok_or("thiếu exec.program")?;
    if !program_ok(program) {
        return Err(format!("exec.program \"{program}\" có đuôi không được phép (chỉ .exe/.bat/.cmd; script khác chạy qua trình thông dịch + prefix)"));
    }
    check_params(o, "args")?;
    check_params(o, "flags")?;

    let mut c = o.clone();
    for (k, d) in [("title", json!(name)), ("description", json!("")), ("category", json!("life")), ("args", json!([])), ("flags", json!([])), ("admin", json!(false)), ("longRunning", json!(false)), ("dryRun", json!(false))] {
        c.entry(k).or_insert(d);
    }
    let json_mode = exec.get("json").and_then(Value::as_bool).unwrap_or(false);
    if !json_mode {
        // json:false → kết quả luôn là văn bản (hoặc stream khi chạy lâu)
        let long = c.get("longRunning").and_then(Value::as_bool).unwrap_or(false);
        c.insert("output".into(), json!(if long { "stream" } else { "text" }));
    } else {
        c.entry("output").or_insert(json!("text"));
    }
    let mut e = exec.clone();
    e.entry("prefix").or_insert(json!([]));
    e.entry("cwd").or_insert(json!("file"));
    e.entry("json").or_insert(json!(false));
    e.insert("baseDir".into(), json!(base));
    c.insert("exec".into(), Value::Object(e));
    c.insert("source".into(), json!("external"));
    c.insert("file".into(), json!(file));
    Ok(Value::Object(c))
}

/// ① foxkit.json ở gốc, ② ở thư mục con trực tiếp (A→Z), bỏ thư mục ẩn và SKIP_DIRS.
pub fn discover(root: &Path) -> Vec<PathBuf> {
    let mut files = Vec::new();
    let top = root.join("foxkit.json");
    if top.is_file() {
        files.push(top);
    }
    let mut dirs: Vec<PathBuf> = std::fs::read_dir(root)
        .into_iter()
        .flatten()
        .flatten()
        .filter(|e| e.file_type().is_ok_and(|t| t.is_dir()))
        .filter(|e| {
            let n = e.file_name().to_string_lossy().to_lowercase();
            !n.starts_with('.') && !SKIP_DIRS.contains(&n.as_str())
        })
        .map(|e| e.path())
        .collect();
    dirs.sort();
    files.extend(dirs.into_iter().map(|d| d.join("foxkit.json")).filter(|f| f.is_file()));
    files
}

/// Quét một thư mục. `seen` chứa tên lệnh đã có (builtin + các thư mục trước) để bỏ lệnh trùng.
fn scan_folder(path: &str, root: &Path, seen: &mut HashSet<String>, total: &mut usize, now: u64, log: &mut Vec<String>) -> FolderSnap {
    let mut snap = FolderSnap { path: path.into(), scanned_at: now, status: "ok".into(), ..Default::default() };
    let found = discover(root);
    if found.is_empty() {
        log.push(format!("{path}: không thấy foxkit.json nào (chỉ tìm ở gốc và thư mục con trực tiếp)"));
    }
    for f in found {
        let fs_ = f.to_string_lossy().into_owned();
        let skip = |snap: &mut FolderSnap, log: &mut Vec<String>, why: String| {
            snap.skipped += 1;
            log.push(format!("bỏ file {fs_}: {why}"));
        };
        match std::fs::metadata(&f) {
            Ok(m) if m.len() > MAX_FILE_BYTES => {
                skip(&mut snap, log, "lớn hơn 256 KB".into());
                continue;
            }
            Err(e) => {
                skip(&mut snap, log, e.to_string());
                continue;
            }
            _ => {}
        }
        let bytes = match std::fs::read(&f) {
            Ok(b) => b,
            Err(e) => {
                skip(&mut snap, log, e.to_string());
                continue;
            }
        };
        let doc: Value = match serde_json::from_slice(&bytes) {
            Ok(v) => v,
            Err(e) => {
                skip(&mut snap, log, format!("JSON hỏng ({e})"));
                continue;
            }
        };
        if doc.get("schema").and_then(Value::as_i64) != Some(1) {
            skip(&mut snap, log, "schema khác 1".into());
            continue;
        }
        let Some(list) = doc.get("commands").and_then(Value::as_array) else {
            skip(&mut snap, log, "thiếu mảng commands".into());
            continue;
        };
        let base = f.parent().map(|p| p.to_string_lossy().into_owned()).unwrap_or_default();
        let mut fsnap = FileSnap { path: fs_.clone(), sha256: sha256_hex(&bytes), commands: vec![] };
        for c in list {
            let label = c.get("name").and_then(Value::as_str).unwrap_or("?").to_string();
            match parse_command(c, &fs_, &base) {
                Err(e) => log.push(format!("bỏ lệnh \"{label}\" trong {fs_}: {e}")),
                Ok(_) if seen.contains(&label) => {
                    log.push(format!("bỏ lệnh \"{label}\" trong {fs_}: trùng tên với lệnh foxkit hoặc lệnh ngoài đã nạp"));
                }
                Ok(_) if *total >= MAX_COMMANDS => log.push(format!("bỏ lệnh \"{label}\" trong {fs_}: vượt giới hạn {MAX_COMMANDS} lệnh ngoài")),
                Ok(cmd) => {
                    seen.insert(label.clone());
                    *total += 1;
                    log.push(format!("nạp lệnh \"{label}\" từ {fs_}"));
                    fsnap.commands.push(cmd);
                }
            }
        }
        snap.files.push(fsnap);
    }
    snap
}

/// Quét `folders` (theo thứ tự). `only` = chỉ quét các thư mục này, còn lại giữ snapshot cũ.
pub fn scan(folders: &[String], only: Option<&[String]>, prev: &Snapshot, builtin: &HashSet<String>, now: u64) -> (Snapshot, ScanReport, Vec<String>) {
    let mut log = Vec::new();
    let mut seen = builtin.clone();
    let mut total = 0usize;
    let mut next = Snapshot::default();
    let mut report = ScanReport::default();
    let old_of = |p: &str| prev.folders.iter().find(|f| f.path == p).cloned();

    for path in folders {
        let carried = only.is_some_and(|o| !o.contains(path));
        let root = PathBuf::from(crate::config::expand_env(path));
        let snap = if carried || !root.is_dir() {
            let mut s = old_of(path).unwrap_or(FolderSnap { path: path.clone(), ..Default::default() });
            if !carried {
                s.status = "missing".into();
                report.folders_missing.push(path.clone());
                log.push(format!("{path}: không tìm thấy thư mục, giữ nguyên snapshot cũ"));
                report.messages.push(format!("Không tìm thấy thư mục {path} (giữ lệnh cũ)"));
            }
            for n in s.files.iter().flat_map(|f| &f.commands).filter_map(|c| c.get("name").and_then(Value::as_str)) {
                seen.insert(n.to_string());
                total += 1;
            }
            s
        } else {
            let s = scan_folder(path, &root, &mut seen, &mut total, now, &mut log);
            report.skipped_files += s.skipped;
            if s.files.is_empty() && s.skipped == 0 {
                report.messages.push(format!("Không thấy foxkit.json nào trong {path} (chỉ tìm ở gốc và thư mục con trực tiếp)"));
            }
            s
        };
        next.folders.push(snap);
    }

    let (old, new) = (prev.commands(), next.commands());
    report.added = new.keys().filter(|k| !old.contains_key(*k)).count() as u32;
    report.removed = old.keys().filter(|k| !new.contains_key(*k)).count() as u32;
    report.changed = new.iter().filter(|(k, v)| old.get(*k).is_some_and(|o| o != *v)).count() as u32;
    (next, report, log)
}

// ---- lưu trữ ----

pub fn cache_path(dir: &Path) -> PathBuf {
    dir.join("scan-cache.json")
}

pub fn load_snapshot(dir: &Path) -> Snapshot {
    std::fs::read_to_string(cache_path(dir)).ok().and_then(|t| serde_json::from_str(&t).ok()).unwrap_or_default()
}

pub fn save_snapshot(dir: &Path, s: &Snapshot) -> Result<(), String> {
    std::fs::create_dir_all(dir).map_err(|e| e.to_string())?;
    std::fs::write(cache_path(dir), serde_json::to_string_pretty(s).map_err(|e| e.to_string())?).map_err(|e| e.to_string())
}

pub fn write_log(dir: &Path, lines: &[String]) {
    let _ = std::fs::create_dir_all(dir); // log là phụ: lỗi ghi không làm hỏng việc quét
    let _ = std::fs::write(dir.join("scan.log"), lines.join("\n"));
}

pub fn read_log(dir: &Path) -> String {
    std::fs::read_to_string(dir.join("scan.log")).unwrap_or_default()
}

fn trusted_path(dir: &Path) -> PathBuf {
    dir.join("trusted.json")
}

pub fn load_trusted(dir: &Path) -> BTreeMap<String, String> {
    std::fs::read_to_string(trusted_path(dir)).ok().and_then(|t| serde_json::from_str(&t).ok()).unwrap_or_default()
}

/// File khai báo đã được cho phép với đúng nội dung hiện tại (snapshot) chưa?
pub fn is_trusted(dir: &Path, file: &str) -> bool {
    let snap = load_snapshot(dir);
    let cur = snap.folders.iter().flat_map(|f| &f.files).find(|f| f.path == file).map(|f| f.sha256.clone());
    cur.is_some_and(|h| load_trusted(dir).get(file) == Some(&h))
}

pub fn trust(dir: &Path, file: &str) -> Result<(), String> {
    let snap = load_snapshot(dir);
    let h = snap.folders.iter().flat_map(|f| &f.files).find(|f| f.path == file).map(|f| f.sha256.clone()).ok_or("file chưa được quét")?;
    let mut t = load_trusted(dir);
    t.insert(file.to_string(), h);
    std::fs::create_dir_all(dir).map_err(|e| e.to_string())?;
    std::fs::write(trusted_path(dir), serde_json::to_string_pretty(&t).map_err(|e| e.to_string())?).map_err(|e| e.to_string())
}

/// Cùng thư mục (không phân biệt hoa thường, bỏ dấu `\` cuối).
pub fn same_folder(a: &str, b: &str) -> bool {
    let n = |s: &str| s.trim_end_matches(['\\', '/']).replace('/', "\\").to_lowercase();
    n(a) == n(b)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn tmp(name: &str) -> PathBuf {
        let d = std::env::temp_dir().join(format!("foxkit-scan-{name}-{}", std::process::id()));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(&d).unwrap();
        d
    }

    fn write(p: &Path, s: &str) {
        fs::create_dir_all(p.parent().unwrap()).unwrap();
        fs::write(p, s).unwrap();
    }

    fn doc(names: &[&str]) -> String {
        let cmds: Vec<String> = names.iter().map(|n| format!(r#"{{"name":"{n}","title":"T {n}","exec":{{"program":"tool.exe"}}}}"#)).collect();
        format!(r#"{{"schema":1,"commands":[{}]}}"#, cmds.join(","))
    }

    fn builtin() -> HashSet<String> {
        ["port", "uuid"].iter().map(|s| s.to_string()).collect()
    }

    fn names(s: &Snapshot) -> Vec<String> {
        s.all_commands().iter().map(|c| c["name"].as_str().unwrap().to_string()).collect()
    }

    fn run(root: &Path, prev: &Snapshot) -> (Snapshot, ScanReport, Vec<String>) {
        scan(&[root.to_string_lossy().into_owned()], None, prev, &builtin(), 1)
    }

    #[test]
    fn only_standard_locations_are_read() {
        let r = tmp("locations");
        write(&r.join("foxkit.json"), &doc(&["root-cmd"]));
        write(&r.join("tool-a/foxkit.json"), &doc(&["a-cmd"]));
        write(&r.join("tool-b/foxkit.json"), &doc(&["b-cmd"]));
        write(&r.join("a/b/foxkit.json"), &doc(&["deep"]));
        write(&r.join("tool-c/x.foxkit.json"), &doc(&["wrong-name"]));
        write(&r.join("tool-c/foxkit.config.json"), &doc(&["wrong-name-2"]));
        for skip in ["node_modules", ".git", "bin", "obj", ".hidden"] {
            write(&r.join(skip).join("foxkit.json"), &doc(&["skipped"]));
        }
        let (s, rep, _) = run(&r, &Snapshot::default());
        assert_eq!(names(&s), ["root-cmd", "a-cmd", "b-cmd"]);
        assert_eq!((rep.added, rep.changed, rep.removed), (3, 0, 0));
    }

    #[test]
    fn choosing_a_tool_folder_uses_root_file() {
        let r = tmp("toolroot");
        write(&r.join("tool-a/foxkit.json"), &doc(&["a-cmd"]));
        let (s, _, _) = run(&r.join("tool-a"), &Snapshot::default());
        assert_eq!(names(&s), ["a-cmd"]);
    }

    #[test]
    fn empty_folder_reports_message() {
        let r = tmp("empty");
        let (_, rep, _) = run(&r, &Snapshot::default());
        assert!(rep.messages[0].starts_with("Không thấy foxkit.json nào trong"));
    }

    #[test]
    fn rescan_reports_added_changed_removed_and_snapshot_matches_disk() {
        let r = tmp("rescan");
        write(&r.join("a/foxkit.json"), &doc(&["one", "two"]));
        let (s1, _, _) = run(&r, &Snapshot::default());
        // sửa "one", xóa "two", thêm file mới
        write(&r.join("a/foxkit.json"), &doc(&["one"]).replace("T one", "T changed"));
        write(&r.join("b/foxkit.json"), &doc(&["three"]));
        let (s2, rep, _) = run(&r, &s1);
        assert_eq!((rep.added, rep.changed, rep.removed), (1, 1, 1));
        assert_eq!(names(&s2), ["one", "three"]);
        // không quét thì snapshot cũ không đổi dù đĩa đổi (hàm scan là thứ duy nhất đổi snapshot)
        write(&r.join("c/foxkit.json"), &doc(&["four"]));
        assert_eq!(names(&s2), ["one", "three"]);
    }

    #[test]
    fn missing_folder_keeps_old_snapshot() {
        let r = tmp("missing");
        write(&r.join("a/foxkit.json"), &doc(&["one"]));
        let (s1, _, _) = run(&r, &Snapshot::default());
        fs::remove_dir_all(&r).unwrap();
        let (s2, rep, _) = run(&r, &s1);
        assert_eq!(names(&s2), ["one"]);
        assert_eq!(s2.folders[0].status, "missing");
        assert_eq!(rep.folders_missing.len(), 1);
        assert_eq!((rep.added, rep.removed), (0, 0));
    }

    #[test]
    fn bad_files_and_commands_are_dropped_without_hurting_others() {
        let r = tmp("bad");
        write(&r.join("a/foxkit.json"), "{ not json");
        write(&r.join("b/foxkit.json"), r#"{"schema":2,"commands":[]}"#);
        write(&r.join("c/foxkit.json"), &doc(&["good"]));
        write(
            &r.join("d/foxkit.json"),
            r#"{"schema":1,"commands":[
                {"name":"no-exec","title":"x"},
                {"name":"Bad Name","exec":{"program":"x.exe"}},
                {"name":"script","exec":{"program":"run.ps1"}},
                {"name":"port","exec":{"program":"x.exe"}},
                {"name":"good","exec":{"program":"x.exe"}},
                {"name":"ok-ps","exec":{"program":"powershell","prefix":["-File","a.ps1"]}}
            ]}"#,
        );
        let (s, rep, log) = run(&r, &Snapshot::default());
        assert_eq!(names(&s), ["good", "ok-ps"]);
        assert_eq!(rep.skipped_files, 2);
        let text = log.join("\n");
        for needle in ["JSON hỏng", "schema khác 1", "thiếu khối exec", "sai quy tắc", "đuôi không được phép", "trùng tên"] {
            assert!(text.contains(needle), "log thiếu: {needle}");
        }
    }

    #[test]
    fn json_false_forces_text_output_and_sets_basedir() {
        let r = tmp("output");
        write(&r.join("foxkit.json"), r#"{"schema":1,"commands":[{"name":"hello","output":"kv","exec":{"program":"cmd"}}]}"#);
        let (s, _, _) = run(&r, &Snapshot::default());
        let c = &s.all_commands()[0];
        assert_eq!(c["output"], "text");
        assert_eq!(c["source"], "external");
        assert_eq!(c["exec"]["cwd"], "file");
        assert_eq!(c["exec"]["baseDir"].as_str().unwrap(), r.to_string_lossy());
    }

    #[test]
    fn only_scans_selected_folder() {
        let (a, b) = (tmp("only-a"), tmp("only-b"));
        write(&a.join("foxkit.json"), &doc(&["from-a"]));
        write(&b.join("foxkit.json"), &doc(&["from-b"]));
        let both = vec![a.to_string_lossy().into_owned(), b.to_string_lossy().into_owned()];
        let (s1, _, _) = scan(&both, None, &Snapshot::default(), &builtin(), 1);
        write(&a.join("foxkit.json"), &doc(&["from-a", "new-a"]));
        write(&b.join("foxkit.json"), &doc(&["from-b", "new-b"]));
        let (s2, rep, _) = scan(&both, Some(&both[..1]), &s1, &builtin(), 2);
        assert_eq!(names(&s2), ["from-a", "new-a", "from-b"]);
        assert_eq!(rep.added, 1);
    }

    #[test]
    fn trust_is_bound_to_file_hash() {
        let r = tmp("trust");
        let dir = tmp("trust-data");
        write(&r.join("foxkit.json"), &doc(&["hello"]));
        let (s, _, _) = run(&r, &Snapshot::default());
        save_snapshot(&dir, &s).unwrap();
        let file = r.join("foxkit.json").to_string_lossy().into_owned();
        assert!(!is_trusted(&dir, &file));
        trust(&dir, &file).unwrap();
        assert!(is_trusted(&dir, &file));
        // file đổi nội dung + quét lại → hash khác → hỏi lại
        write(&r.join("foxkit.json"), &doc(&["hello", "more"]));
        let (s2, _, _) = run(&r, &s);
        save_snapshot(&dir, &s2).unwrap();
        assert!(!is_trusted(&dir, &file));
    }

    #[test]
    fn same_folder_ignores_case_and_trailing_slash() {
        assert!(same_folder(r"D:\TranCN\foxkit\", r"d:\trancn\foxkit"));
        assert!(!same_folder(r"D:\a", r"D:\b"));
    }

    // ---- JSON Schema: bộ kiểm tối giản (type, required, properties, items, enum, const, pattern-free, $ref) ----
    fn check(root: &Value, schema: &Value, v: &Value, at: &str) -> Result<(), String> {
        if let Some(r) = schema.get("$ref").and_then(Value::as_str) {
            let target = root.pointer(r.trim_start_matches('#')).ok_or(format!("$ref lạ {r}"))?;
            return check(root, target, v, at);
        }
        if let Some(c) = schema.get("const") {
            if c != v {
                return Err(format!("{at}: phải bằng {c}"));
            }
        }
        if let Some(e) = schema.get("enum").and_then(Value::as_array) {
            if !e.contains(v) {
                return Err(format!("{at}: ngoài enum"));
            }
        }
        if let Some(t) = schema.get("type").and_then(Value::as_str) {
            let ok = match t {
                "object" => v.is_object(),
                "array" => v.is_array(),
                "string" => v.is_string(),
                "boolean" => v.is_boolean(),
                "number" | "integer" => v.is_number(),
                _ => true,
            };
            if !ok {
                return Err(format!("{at}: sai kiểu, cần {t}"));
            }
        }
        if let (Some(req), Some(o)) = (schema.get("required").and_then(Value::as_array), v.as_object()) {
            for k in req.iter().filter_map(Value::as_str) {
                if !o.contains_key(k) {
                    return Err(format!("{at}: thiếu {k}"));
                }
            }
        }
        if let (Some(props), Some(o)) = (schema.get("properties").and_then(Value::as_object), v.as_object()) {
            for (k, s) in props {
                if let Some(x) = o.get(k) {
                    check(root, s, x, &format!("{at}/{k}"))?;
                }
            }
        }
        if let (Some(items), Some(a)) = (schema.get("items"), v.as_array()) {
            for (i, x) in a.iter().enumerate() {
                check(root, items, x, &format!("{at}/{i}"))?;
            }
            if let Some(max) = schema.get("maxItems").and_then(Value::as_u64) {
                if a.len() as u64 > max {
                    return Err(format!("{at}: tối đa {max} phần tử"));
                }
            }
        }
        Ok(())
    }

    fn schema() -> Value {
        serde_json::from_str(include_str!("../../schema/foxkit-command.schema.json")).unwrap()
    }

    #[test]
    fn schema_file_is_valid_json_and_accepts_good_sample() {
        let s = schema();
        let sample: Value = serde_json::from_str(
            r#"{"$schema":"x","schema":1,"commands":[{"name":"resize","title":"Đổi cỡ ảnh","description":"d","category":"life",
              "args":[{"name":"input","type":"file","required":true,"label":"Ảnh"}],
              "flags":[{"name":"width","type":"int","min":16,"max":8000,"default":1024,"label":"Rộng"}],
              "admin":false,"output":"text","examples":[{"cmd":"resize a.jpg","desc":"d"}],
              "exec":{"program":"bin\\resize.exe","prefix":[],"cwd":"file","json":false}}]}"#,
        )
        .unwrap();
        check(&s, &s, &sample, "").unwrap();
    }

    #[test]
    fn schema_rejects_bad_samples() {
        let s = schema();
        for bad in [
            r#"{"schema":2,"commands":[]}"#,
            r#"{"schema":1}"#,
            r#"{"schema":1,"commands":[{"name":"x"}]}"#,
            r#"{"schema":1,"commands":[{"name":"x","exec":{"program":"a"},"args":[{"name":"a","type":"weird"}]}]}"#,
            r#"{"schema":1,"commands":[{"name":"x","exec":{"program":"a"},"category":"nope"}]}"#,
        ] {
            assert!(check(&s, &s, &serde_json::from_str(bad).unwrap(), "").is_err(), "{bad}");
        }
    }

    #[test]
    fn everything_the_scanner_accepts_passes_the_schema() {
        let s = schema();
        let d: Value = serde_json::from_str(&doc(&["a", "b"])).unwrap();
        check(&s, &s, &d, "").unwrap();
    }
}
