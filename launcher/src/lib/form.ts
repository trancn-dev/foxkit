import { label, type Command } from "./manifest";
import { parseLine, toArgs, type ArgError } from "./resolve";

export interface FormValues {
  /** Giá trị theo tên arg; arg `multiple` là mảng (mỗi dòng một giá trị). */
  args: Record<string, string | string[]>;
  /** Cờ bool: true/false; còn lại là chuỗi (rỗng = bỏ qua). */
  flags: Record<string, string | boolean>;
}

export interface FormResult {
  args: string[];
  error?: ArgError;
}

/** Giá trị form → mảng args. Dùng đúng `parseLine` như gõ tay nên cùng quy tắc và cùng thông báo lỗi. */
export function formToArgs(cmd: Command, v: FormValues): FormResult {
  const tokens = cmd.name.split(" ");
  let gap: string | null = null;
  for (const p of cmd.args) {
    const raw = v.args[p.name];
    const vals = (Array.isArray(raw) ? raw : [raw ?? ""]).map((s) => s.trim()).filter((s) => s !== "");
    if (vals.length === 0) {
      gap ??= label(p);
      continue;
    }
    // đã có arg trống phía trước mà arg sau lại có giá trị: không thể bỏ qua vị trí giữa chừng
    if (gap) return { args: [], error: { code: "ARG_INVALID", message: `Thiếu ${gap}`, hint: cmd.name } };
    tokens.push(...vals);
  }
  for (const f of cmd.flags) {
    const raw = v.flags[f.name];
    if (f.type === "bool") {
      if (raw === true) tokens.push(`--${f.name}`);
    } else if (typeof raw === "string" && raw.trim() !== "") {
      tokens.push(`--${f.name}`, raw.trim());
    }
  }
  const parsed = parseLine(tokens, [cmd]);
  if (!parsed) return { args: [] };
  return { args: toArgs(parsed), error: parsed.error };
}
