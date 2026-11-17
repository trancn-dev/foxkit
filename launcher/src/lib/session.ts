// Bộ điều khiển dùng chung cho ô nhanh và giao diện chính: nạp manifest, chạy lệnh, cập nhật luồng chat.
import { Channel, invoke } from "@tauri-apps/api/core";
import { emit, listen } from "@tauri-apps/api/event";
import { readText, writeText } from "@tauri-apps/plugin-clipboard-manager";
import { computed, ref, shallowRef } from "vue";
import { exampleFor, ghost, type Ghost } from "./ghost";
import { copyForResult, interpret, STREAM_MAX, type Result, type RunResult } from "./interpret";
import { normalizeManifest, type Command, type Example, type Manifest } from "./manifest";
import { expand, type ContextError } from "./placeholders";
import { childrenOf, findCommand, parseLine, syntax, toArgs, type ArgError, type ParsedLine } from "./resolve";
import { search } from "./fuzzy";
import { preferredFor, type AppHint } from "./appcmds";
import { ThreadStore, quickView, type ThreadBackend, type ThreadItem } from "./thread";
import { quoteToken, tokenize } from "./tokenize";

export interface Ctx {
  app: string;
  exe: string;
  title: string;
  dir: string | null;
  files: string[];
  source: "explorer" | "vscode" | "generic" | "none";
}

export interface AppConfig {
  hotkey: string;
  foxkitPath: string;
  autostart: boolean;
  autoCopy: boolean;
  context: boolean;
  folders: string[];
  appHints: AppHint[];
}

export interface AppError {
  code: string;
  message: string;
  hint?: string;
}

/** Kết quả gửi lệnh: `ran` = đã qua validate/xác nhận và bắt đầu chạy; `done` xong khi lệnh kết thúc. */
export interface Submitted {
  ran: boolean;
  done: Promise<{ copied: boolean }>;
}
const notRun = (): Submitted => ({ ran: false, done: Promise.resolve({ copied: false }) });

export interface Prompt {
  text: string;
  resolve: (yes: boolean) => void;
}

const NO_CTX: Ctx = { app: "", exe: "", title: "", dir: null, files: [], source: "none" };

/** Lệnh nội bộ của launcher (bắt đầu bằng `:`), có trong gợi ý như lệnh thường (launcher.md 6.9). */
const internal = (name: string, title: string, description: string, args: Command["args"] = [], examples: Example[] = []): Command => ({
  name, title, description, category: "launcher", args, flags: [], admin: false, longRunning: false, dryRun: false, output: "text", source: "launcher", examples,
});
export const INTERNAL_COMMANDS: Command[] = [
  internal(":add", "Thêm thư mục lệnh", "Chọn thư mục chứa foxkit.json rồi quét", [{ name: "path", type: "dir", label: "Thư mục" }], [{ cmd: ":add @dir", desc: "Thêm thư mục đang mở" }]),
  internal(":scan", "Quét lại thư mục lệnh", "Đọc lại mọi thư mục lệnh ngoài và cập nhật"),
  internal(":folders", "Danh sách thư mục lệnh", "Liệt kê thư mục lệnh ngoài và trạng thái"),
  internal(":forget", "Bỏ thư mục lệnh", "Bỏ thư mục khỏi danh sách và xóa lệnh của nó", [{ name: "path", type: "dir", required: true, label: "Thư mục" }]),
  internal(":clear", "Xóa cuộc chat", "Xóa luồng chat ở cả hai cửa sổ"),
  internal(":open", "Mở giao diện chính", "Hiện cửa sổ chính"),
];

const backend: ThreadBackend = {
  list: () => invoke<ThreadItem[]>("thread_list"),
  add: (item) => invoke<number>("thread_add", { item }),
  update: (id, item) => invoke<void>("thread_update", { id, item }),
  remove: (id) => invoke<void>("thread_remove", { id }),
  clear: () => invoke<void>("thread_clear"),
  onChanged: (cb) => listen("thread-changed", cb),
};

function asError(e: unknown): AppError {
  const x = e as Partial<AppError>;
  return x && typeof x === "object" && x.code && x.message ? (x as AppError) : { code: "INTERNAL", message: String((e as Error)?.message ?? e) };
}

interface ExecInfo {
  program: string;
  prefix: string[];
  json: boolean;
  cwd: string;
  baseDir: string;
}

export class Session {
  manifest = shallowRef<Manifest | null>(null);
  loadError = ref<AppError | null>(null);
  config = ref<AppConfig>({ hotkey: "Ctrl+Alt+Space", foxkitPath: "", autostart: true, autoCopy: false, context: true, folders: [], appHints: [] });
  ctx = ref<Ctx>(NO_CTX);
  clip = ref<string | null>(null);
  prompt = ref<Prompt | null>(null);
  error = ref<AppError | null>(null);
  busy = ref<{ runId?: number } | null>(null);
  items = ref<ThreadItem[]>([]);
  /** Id mục luồng chat của lần chạy gần nhất (form dùng để hiện kết quả ngay dưới). */
  lastItemId = ref(0);
  private store = new ThreadStore(backend);

  /** Cửa sổ chính không có ngữ cảnh app đang focus (launcher.md 6.10). */
  constructor(readonly windowKind: "bar" | "main") {}

  commands = computed<Command[]>(() => [...(this.manifest.value?.commands ?? []), ...INTERNAL_COMMANDS]);
  quick = computed(() => quickView(this.items.value));

  async init(): Promise<void> {
    this.store.subscribe((items) => (this.items.value = items));
    await this.store.start();
    await this.reload();
  }

  /** Đọc lại config + manifest (tray "Tải lại", sau `:scan`...). */
  async reload(): Promise<void> {
    try {
      this.config.value = await invoke<AppConfig>("get_config");
      this.manifest.value = normalizeManifest(await invoke("load_manifest"));
      this.loadError.value = null;
    } catch (e) {
      this.loadError.value = asError(e);
    }
  }

  /** Gọi mỗi lần cửa sổ ô nhanh hiện: ngữ cảnh đã chụp ở Rust + clipboard. */
  async refreshContext(): Promise<void> {
    this.ctx.value = NO_CTX;
    this.clip.value = null;
    if (!this.config.value.context) return;
    try {
      if (this.windowKind === "bar") this.ctx.value = await invoke<Ctx>("get_context");
      const t = (await readText())?.split(/\r?\n/)[0]?.slice(0, 500);
      this.clip.value = t || null; // chỉ giữ trong bộ nhớ
    } catch {
      this.clip.value = null; // clipboard rỗng/không phải văn bản
    }
  }

  // ---- gợi ý ----
  ghostFor(input: string): Ghost | null {
    return ghost(input, this.commands.value, { dir: this.ctx.value.dir, files: this.ctx.value.files, clip: this.clip.value });
  }
  exampleFor(input: string, pick?: (n: number) => number): Example | null {
    return exampleFor(input, this.commands.value, pick, preferredFor(this.ctx.value.exe, this.config.value.appHints));
  }
  suggestions(input: string): Command[] {
    const { tokens, trailingSpace } = tokenize(input);
    if (tokens.length === 0) {
      // Ô trống: giữ ô nhanh gọn, trừ khi app đang focus có bộ lệnh riêng (vd. trình duyệt).
      const names = preferredFor(this.ctx.value.exe, this.config.value.appHints);
      return names.map((n) => this.commands.value.find((c) => c.name === n)).filter((c): c is Command => !!c);
    }
    if (findCommand(tokens, this.commands.value) && (trailingSpace || tokens.length > 1)) return [];
    const kids = childrenOf(tokens, this.commands.value);
    return kids.length ? kids.slice(0, 8) : search(tokens.join(" "), this.commands.value, 8);
  }
  syntaxFor(input: string): string | null {
    const f = findCommand(tokenize(input).tokens, this.commands.value);
    return f ? syntax(f.cmd) : null;
  }

  // ---- hỏi xác nhận ngay trong ô nhập ----
  ask(text: string): Promise<boolean> {
    return new Promise((resolve) => (this.prompt.value = { text, resolve }));
  }
  answer(yes: boolean): void {
    const p = this.prompt.value;
    this.prompt.value = null;
    p?.resolve(yes);
  }

  // ---- chạy ----
  private fail(e: AppError): false {
    this.error.value = e;
    return false;
  }

  /** Chạy một dòng gõ vào. Trả true nếu lệnh được chạy (kể cả khi kết quả là lỗi của lệnh). `copyAfter`: Shift+Enter. */
  async submit(line: string, opts: { copyAfter?: boolean } = {}): Promise<Submitted> {
    this.error.value = null;
    const raw = tokenize(line).tokens;
    if (raw.length === 0) return notRun();
    if (raw[0].startsWith(":")) return { ran: await this.runInternal(raw, line), done: Promise.resolve({ copied: false }) };

    let tokens: string[];
    try {
      tokens = expand(raw, { enabled: this.config.value.context, dir: this.ctx.value.dir, files: this.ctx.value.files, clip: this.clip.value });
    } catch (e) {
      this.fail(this.contextError(e as ContextError));
      return notRun();
    }
    const parsed = parseLine(tokens, this.commands.value);
    if (!parsed) {
      const kids = childrenOf(tokens, this.commands.value);
      this.fail({ code: "UNKNOWN_COMMAND", message: `Không có lệnh "${tokens.join(" ")}"`, hint: kids.length ? `Lệnh con: ${kids.map((k) => k.name).join(", ")}` : "Chọn một lệnh trong danh sách gợi ý" });
      return notRun();
    }
    if (parsed.error) {
      this.fail(parsed.error);
      return notRun();
    }
    return this.execute(parsed, line, opts.copyAfter ?? false);
  }

  /** Chạy args đã dựng từ form: parse + xác nhận + dryRun + trust như gõ tay. */
  async submitArgs(args: string[]): Promise<Submitted> {
    this.error.value = null;
    const parsed = parseLine(args, this.commands.value);
    if (!parsed) return notRun();
    if (parsed.error) {
      this.fail(parsed.error);
      return notRun();
    }
    return this.execute(parsed, `foxkit ${args.map(quoteToken).join(" ")}`, false);
  }

  private contextError(e: ContextError): AppError {
    if (this.windowKind === "main" && e.message.includes("thư mục")) return { ...e, hint: "Dùng ô lệnh nhanh bằng phím tắt khi đang ở Explorer/VS Code, hoặc gõ đường dẫn trực tiếp" };
    return e;
  }

  private async execute(parsed: ParsedLine, line: string, copyAfter: boolean): Promise<Submitted> {
    const cmd = parsed.cmd;
    const none = notRun();
    if (this.busy.value) {
      if (!(await this.ask("Đang có lệnh chạy. Enter = Dừng và chạy lệnh mới · Esc = Thôi"))) return none;
      if (this.busy.value.runId) await invoke("stop", { runId: this.busy.value.runId });
    }
    if (cmd.source === "external" && cmd.file) {
      const file = cmd.file;
      if (!(await invoke<boolean>("trust_check", { file }))) {
        const prog = (cmd as Command & { exec?: ExecInfo }).exec?.program ?? "?";
        if (!(await this.ask(`Cho phép chạy "${prog}" từ ${file}? Enter = Cho phép và nhớ · Esc = Không`))) {
          this.fail({ code: "EXTERNAL_DENIED", message: "Chưa cho phép chạy lệnh ngoài này", hint: "Chạy lại và chọn Cho phép" });
          return none;
        }
        await invoke("trust_allow", { file });
      }
    }
    const asks = [cmd.confirm, ...parsed.flags.map((f) => cmd.flags.find((x) => x.name === f.name)?.confirm)].filter(Boolean) as string[];
    for (const q of asks) if (!(await this.ask(`${q} Enter = Có · Esc = Không`))) return none;

    const args = toArgs(parsed);
    const preview = cmd.dryRun && !parsed.flags.some((f) => f.name === "apply");
    return { ran: true, done: this.run(cmd, args, line, { preview, copyAfter }) };
  }

  /** Chạy args đã validate, ghi kết quả vào luồng chat. Dùng cho nút Áp dụng / Chạy admin / form. */
  async run(cmd: Command, args: string[], line: string, opts: { preview?: boolean; copyAfter?: boolean; elevated?: boolean } = {}): Promise<{ copied: boolean }> {
    const id = await this.store.begin(line);
    this.lastItemId.value = id;
    const exec = (cmd as Command & { exec?: ExecInfo }).exec;
    const nameParts = cmd.source === "external" ? cmd.name.split(" ").length : 0;
    const sendArgs = nameParts ? args.slice(nameParts) : args; // lệnh ngoài không nhận tên lệnh
    const cwd = exec ? (exec.cwd === "context" && this.ctx.value.dir ? this.ctx.value.dir : exec.baseDir) : this.ctx.value.dir;
    const spec = exec ? { program: exec.program, prefix: exec.prefix, json: exec.json, baseDir: exec.baseDir } : undefined;
    const stream = cmd.longRunning || (!!cmd.longRunningFlag && args.includes(`--${cmd.longRunningFlag}`));
    let result: Result;
    try {
      if (stream) {
        result = await this.runStream(id, line, sendArgs, cwd, spec);
      } else if (opts.elevated) {
        this.busy.value = {};
        result = interpret(await invoke<RunResult>("run_elevated", { args: sendArgs, cwd }), cmd, args);
      } else {
        this.busy.value = {};
        result = interpret(await invoke<RunResult>("run", { args: sendArgs, cwd, exec: spec }), cmd, args);
      }
    } catch (e) {
      const a = asError(e);
      result = { kind: "error", code: a.code, message: a.message, hint: a.hint };
    } finally {
      this.busy.value = null;
    }
    if (result.kind === "ok" && opts.preview) result = { ...result, dryRun: true, applyArgs: [...args, "--apply"] };
    await this.store.finish(id, line, result.kind === "error" ? "error" : "done", result);
    const copyable = result.kind === "ok" && !opts.preview && (this.config.value.autoCopy || opts.copyAfter);
    let copied = false;
    if (copyable) copied = await this.copyResult(result);
    return { copied };
  }

  private async runStream(id: number, line: string, args: string[], cwd: string | null, exec: unknown): Promise<Result> {
    const lines: string[] = [];
    const ch = new Channel<{ kind: "Line" | "Stderr" | "Exit"; text?: string; code?: number }>();
    let exit: number | undefined;
    let dirty = false;
    const flush = (running: boolean) => {
      dirty = false;
      return this.store.finish(id, line, "done", { kind: "stream", lines: lines.slice(-STREAM_MAX), running, exit } satisfies Result);
    };
    const timer = setInterval(() => dirty && void flush(true), 250);
    const done = new Promise<void>((resolve) => {
      ch.onmessage = (m) => {
        if (m.kind === "Exit") {
          exit = m.code;
          resolve();
        } else {
          lines.push(m.text ?? "");
          if (lines.length > STREAM_MAX * 2) lines.splice(0, lines.length - STREAM_MAX);
          dirty = true;
        }
      };
    });
    try {
      const runId = await invoke<number>("run_stream", { args, cwd, exec, onLine: ch });
      this.busy.value = { runId };
      await flush(true);
      await done;
    } finally {
      clearInterval(timer);
    }
    return { kind: "stream", lines: lines.slice(-STREAM_MAX), running: false, exit };
  }

  async stop(): Promise<void> {
    const id = this.busy.value?.runId;
    if (id) await invoke("stop", { runId: id });
  }

  /** Xóa đúng một mục khỏi luồng chat. */
  removeItem(id: number): Promise<void> {
    return this.store.remove(id);
  }

  /** Xóa toàn bộ luồng chat ở cả hai cửa sổ (cùng việc với lệnh `:clear`). */
  clearChat(): Promise<void> {
    return this.store.clear();
  }

  async copyResult(r: Result): Promise<boolean> {
    const text = copyForResult(r);
    if (text === null) return false;
    await writeText(text);
    return true;
  }

  /** Nút "Chạy lại với quyền admin" / "Áp dụng" trên một kết quả. */
  async rerun(item: ThreadItem, args: string[], elevated: boolean): Promise<void> {
    const cmd = findCommand(args, this.commands.value)?.cmd;
    if (cmd) await this.run(cmd, args, `${item.line}${elevated ? " (admin)" : " --apply"}`, { elevated });
  }

  // ---- lệnh nội bộ ----
  private async info(line: string, text: string, lines?: string[]): Promise<void> {
    const id = await this.store.begin(line);
    await this.store.finish(id, line, "done", { kind: "info", text, lines } satisfies Result);
  }

  private async runInternal(tokens: string[], line: string): Promise<boolean> {
    const name = tokens[0].toLowerCase();
    const arg = tokens.slice(1);
    const expandPath = (t: string) => {
      if (t === "@dir") {
        if (!this.ctx.value.dir) throw this.contextError({ code: "CONTEXT_MISSING", message: "Không xác định được thư mục hiện tại", hint: "" });
        return this.ctx.value.dir;
      }
      return t;
    };
    try {
      switch (name) {
        case ":clear":
          await this.store.clear();
          return true;
        case ":open":
          await invoke("show_main", { tab: "chat" });
          return true;
        case ":folders": {
          const f = await invoke<{ path: string; files: number; commands: number; skipped: number; scanned_at: number; status: string }[]>("folders_info");
          const rows = f.map((x) => `${x.path} · ${x.files} file · ${x.commands} lệnh · ${x.status}${x.scanned_at ? " · quét " + new Date(x.scanned_at * 1000).toLocaleString("vi-VN") : ""}`);
          await this.info(line, f.length ? `${f.length} thư mục lệnh` : "Chưa có thư mục lệnh nào. Dùng :add để thêm.", rows);
          return true;
        }
        case ":scan":
          await this.report(line, await invoke("scan", {}));
          return true;
        case ":add": {
          const path = arg[0] ? expandPath(arg[0]) : await this.pickFolder();
          if (!path) return false;
          await this.report(line, await invoke("add_folder", { path }));
          return true;
        }
        case ":forget": {
          if (!arg[0]) return this.fail({ code: "ARG_INVALID", message: "Thiếu Thư mục", hint: ":forget <đường dẫn>" });
          const path = expandPath(arg[0]);
          const removed = await invoke<boolean>("forget_folder", { path });
          await this.info(line, removed ? `Đã bỏ ${path}` : `Không có thư mục ${path} trong danh sách`);
          await this.reload();
          return true;
        }
        default:
          return this.fail({ code: "UNKNOWN_COMMAND", message: `Không có lệnh "${name}"`, hint: "Lệnh nội bộ: :add :scan :folders :forget :clear :open" });
      }
    } catch (e) {
      return this.fail(asError(e));
    }
  }

  async pickFolder(): Promise<string | null> {
    const { open } = await import("@tauri-apps/plugin-dialog");
    await invoke("suspend_autohide", { on: true });
    try {
      const p = await open({ directory: true, multiple: false });
      return typeof p === "string" ? p : null;
    } finally {
      await invoke("suspend_autohide", { on: false });
    }
  }

  async report(line: string, r: { added: number; changed: number; removed: number; skipped_files: number; folders_missing: string[]; messages: string[] }): Promise<void> {
    const n = (await invoke<unknown[]>("folders_info")).length;
    const head = `Quét xong ${n} thư mục: +${r.added} lệnh mới, ~${r.changed} đổi, −${r.removed} không còn, ${r.skipped_files} file bị bỏ${r.skipped_files ? " (xem scan.log)" : ""}`;
    await this.info(line, head, r.messages);
    await this.reload();
  }

  /** Điền sẵn ô nhập từ cửa sổ kia (nút Mở rộng / Thu nhỏ). */
  sendDraft(to: "bar" | "main", text: string): Promise<void> {
    return emit("draft", { to, text });
  }
  onDraft(cb: (text: string) => void): Promise<() => void> {
    return listen<{ to: string; text: string }>("draft", (e) => e.payload.to === this.windowKind && cb(e.payload.text));
  }
}
