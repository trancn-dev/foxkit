// Kiểu của manifest schema 1 (foundation mục 9.1) + load qua Rust.
export type ParamType = "string" | "text" | "int" | "bool" | "enum" | "file" | "dir" | "list";

export interface Param {
  name: string;
  type: ParamType;
  required?: boolean;
  default?: string | number | boolean;
  min?: number;
  max?: number;
  options?: string[];
  multiple?: boolean;
  allowDir?: boolean;
  separator?: string;
  label?: string;
  confirm?: string;
}

export interface Example {
  cmd: string;
  desc: string;
}

export type OutputKind = "kv" | "table" | "text" | "stream";

export interface Command {
  name: string;
  title: string;
  description: string;
  category: string;
  args: Param[];
  flags: Param[];
  admin: boolean;
  confirm?: string;
  longRunning: boolean;
  longRunningFlag?: string;
  dryRun: boolean;
  output: OutputKind;
  examples?: Example[];
  source?: "builtin" | "external" | "launcher";
  file?: string;
}

export interface Manifest {
  schema: number;
  commands: Command[];
}

export function label(p: Param): string {
  return p.label || p.name;
}

/** Kiểm schema và chuẩn hóa mảng thiếu (args/flags) để phần còn lại khỏi kiểm null. */
export function normalizeManifest(raw: unknown): Manifest {
  const m = raw as Manifest;
  if (!m || typeof m !== "object" || !Array.isArray(m.commands)) {
    throw { code: "MANIFEST_FAILED", message: "Không đọc được danh sách lệnh từ foxkit", hint: "Chạy `foxkit manifest --json` trong terminal để xem lỗi" };
  }
  if (m.schema !== 1) {
    throw { code: "MANIFEST_UNSUPPORTED", message: "foxkit dùng manifest phiên bản mới hơn launcher", hint: "Cập nhật launcher" };
  }
  for (const c of m.commands) {
    c.args ??= [];
    c.flags ??= [];
    c.source ??= "builtin";
  }
  return m;
}
