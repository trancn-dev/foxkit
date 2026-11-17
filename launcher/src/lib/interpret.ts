import type { Command, OutputKind } from "./manifest";

export interface RunResult {
  exit_code: number;
  stdout: string;
  stderr: string;
}

/** Kết quả đã chuẩn hóa, lưu trong luồng chat (JSON thuần) để cả hai cửa sổ cùng hiển thị. */
export type Result =
  | { kind: "ok"; output: OutputKind; data: unknown; raw: string; dryRun?: boolean; applyArgs?: string[] }
  | { kind: "error"; code: string; message: string; hint?: string; exit?: number; canAdmin?: boolean; args?: string[]; detail?: string }
  | { kind: "stream"; lines: string[]; running: boolean; exit?: number }
  | { kind: "info"; text: string; lines?: string[] };

export const STREAM_MAX = 1000;

function tryJson(s: string): unknown {
  try {
    return JSON.parse(s);
  } catch {
    return undefined;
  }
}

const bad = (res: RunResult): Result => ({
  kind: "error",
  code: "BAD_OUTPUT",
  message: "foxkit trả kết quả không đúng định dạng",
  hint: "Báo lỗi kèm nội dung bên dưới",
  detail: [res.stdout, res.stderr].filter(Boolean).join("\n"),
});

/** Đọc kết quả theo exit code (launcher.md 6.6 bước 7) và quy tắc lệnh ngoài (6.9). */
export function interpret(res: RunResult, cmd: Command, args: string[] = []): Result {
  const external = cmd.source === "external";
  const exec = (cmd as Command & { exec?: { json?: boolean } }).exec;
  if (external && !exec?.json) {
    if (res.exit_code === 0) return { kind: "ok", output: "text", data: res.stdout.replace(/\s+$/, ""), raw: res.stdout };
    return {
      kind: "error",
      code: "EXTERNAL_FAILED",
      message: `Lệnh ${cmd.name} kết thúc với mã ${res.exit_code}`,
      hint: "Xem nội dung lỗi bên dưới",
      exit: res.exit_code,
      detail: (res.stderr || res.stdout).trim(),
    };
  }
  if (res.exit_code === 0) {
    const env = tryJson(res.stdout) as { ok?: boolean; data?: unknown } | undefined;
    if (!env || env.ok !== true || !("data" in env)) return bad(res);
    return { kind: "ok", output: cmd.output, data: env.data, raw: res.stdout };
  }
  const env = tryJson(res.stderr) as { ok?: boolean; error?: { code?: string; message?: string; hint?: string } } | undefined;
  if (!env || env.ok !== false || !env.error?.message) return bad(res);
  return {
    kind: "error",
    code: env.error.code ?? "UNKNOWN",
    message: env.error.message,
    hint: env.error.hint,
    exit: res.exit_code,
    canAdmin: res.exit_code === 3 && !cmd.longRunning,
    args,
  };
}

const cell = (v: unknown): string => (v === null || v === undefined ? "" : typeof v === "object" ? JSON.stringify(v) : String(v)).replace(/[\t\r\n]+/g, " ");

/** Cột của bảng: khóa của phần tử đầu, giữ thứ tự. */
export function columnsOf(items: Record<string, unknown>[]): string[] {
  return items.length ? Object.keys(items[0]) : [];
}

export function tableItems(data: unknown): Record<string, unknown>[] {
  const items = (data as { items?: unknown })?.items;
  return Array.isArray(items) ? (items as Record<string, unknown>[]) : [];
}

/** Quy tắc "copy gì" (launcher.md mục 7). */
export function copyText(r: Extract<Result, { kind: "ok" }>): string {
  const d = r.data;
  if (typeof d === "string") return d;
  if (r.output === "table") {
    const items = tableItems(d);
    const cols = columnsOf(items);
    return [cols.join("\t"), ...items.map((it) => cols.map((c) => cell(it[c])).join("\t"))].join("\n");
  }
  if (d && typeof d === "object" && !Array.isArray(d)) {
    const keys = Object.keys(d);
    if (keys.length === 1) {
      const v = (d as Record<string, unknown>)[keys[0]];
      return typeof v === "string" ? v : typeof v === "object" ? JSON.stringify(v, null, 2) : String(v);
    }
  }
  return JSON.stringify(d, null, 2);
}

/** `uuid` trả `{ items: [...] }` (kv một khóa là mảng chuỗi) → mỗi UUID một dòng sẽ dễ dùng hơn JSON. */
export function copyForResult(r: Result): string | null {
  if (r.kind === "ok") {
    const d = r.data as { items?: unknown };
    if (r.output !== "table" && Array.isArray(d?.items) && d.items.every((x) => typeof x === "string")) return (d.items as string[]).join("\n");
    return copyText(r);
  }
  if (r.kind === "info") return r.text;
  return null;
}
