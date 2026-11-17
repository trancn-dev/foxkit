import { label, type Command, type Param } from "./manifest";

export interface ArgError {
  code: "ARG_INVALID";
  message: string;
  hint: string;
}

export interface FlagUse {
  name: string;
  value: string | true;
}

export interface ParsedLine {
  cmd: Command;
  positional: string[];
  flags: FlagUse[];
  error?: ArgError;
}

/** Cú pháp dạng `port [port] [--kill] [--list]`, dùng cho ArgHint và hint của lỗi. */
export function syntax(cmd: Command): string {
  const a = cmd.args.map((p) => {
    const n = p.multiple ? `${label(p)}...` : label(p);
    return p.required ? `<${n}>` : `[${n}]`;
  });
  const f = cmd.flags.map((p) => (p.type === "bool" ? `[--${p.name}]` : `[--${p.name} <${label(p)}>]`));
  return [cmd.name, ...a, ...f].join(" ");
}

/** Lệnh có `name` là tiền tố dài nhất của tokens; `used` = số token thuộc tên lệnh. */
export function findCommand(tokens: string[], commands: Command[]): { cmd: Command; used: number } | null {
  let best: { cmd: Command; used: number } | null = null;
  for (const c of commands) {
    const parts = c.name.split(" ");
    if (parts.length > tokens.length || (best && parts.length <= best.used)) continue;
    if (parts.every((p, i) => tokens[i].toLowerCase() === p)) best = { cmd: c, used: parts.length };
  }
  return best;
}

/** Các lệnh con của một lệnh cha không có entry (vd `hosts` → hosts add, hosts rm...). */
export function childrenOf(prefix: string[], commands: Command[]): Command[] {
  const p = prefix.map((t) => t.toLowerCase()).join(" ");
  return commands.filter((c) => c.name.startsWith(p + " "));
}

function fail(cmd: Command, message: string): ArgError {
  return { code: "ARG_INVALID", message, hint: syntax(cmd) };
}

/** Kiểm giá trị theo kiểu của tham số; trả thông báo lỗi hoặc null. */
export function checkValue(p: Param, v: string): string | null {
  const l = label(p);
  if (p.type === "int") {
    const ok = /^-?\d+$/.test(v) && (p.min === undefined || +v >= p.min) && (p.max === undefined || +v <= p.max);
    if (ok) return null;
    if (p.min !== undefined && p.max !== undefined) return `${l} phải là số từ ${p.min} đến ${p.max}`;
    return `${l} phải là số nguyên`;
  }
  if (p.type === "enum" && p.options && !p.options.includes(v)) {
    return `${l} phải là một trong: ${p.options.join(", ")}`;
  }
  return null;
}

/** Phân tích dòng đã tách token (không gồm placeholder chưa mở rộng) theo manifest. Trả null nếu chưa khớp lệnh nào. */
export function parseLine(tokens: string[], commands: Command[]): ParsedLine | null {
  const found = findCommand(tokens, commands);
  if (!found) return null;
  const { cmd, used } = found;
  const res: ParsedLine = { cmd, positional: [], flags: [] };
  const rest = tokens.slice(used);
  for (let i = 0; i < rest.length; i++) {
    const t = rest[i];
    if (t === "--json") continue; // launcher luôn tự thêm
    if (t.startsWith("--") && t.length > 2) {
      const eq = t.indexOf("=");
      const name = eq < 0 ? t.slice(2) : t.slice(2, eq);
      const def = cmd.flags.find((f) => f.name === name);
      if (!def) {
        res.error ??= fail(cmd, `Lệnh ${cmd.name} không có cờ --${name}`);
        continue;
      }
      if (def.type === "bool") {
        res.flags.push({ name, value: true });
        continue;
      }
      let value = eq >= 0 ? t.slice(eq + 1) : undefined;
      if (value === undefined) {
        if (i + 1 < rest.length && !rest[i + 1].startsWith("--")) value = rest[++i];
        else {
          res.error ??= fail(cmd, `Cờ --${name} cần một giá trị`);
          continue;
        }
      }
      const bad = checkValue(def, value);
      if (bad) res.error ??= fail(cmd, bad);
      res.flags.push({ name, value });
      continue;
    }
    res.positional.push(t);
  }

  // gán positional vào args theo thứ tự
  cmd.args.forEach((p, idx) => {
    const vals = p.multiple ? res.positional.slice(idx) : res.positional.slice(idx, idx + 1);
    if (p.required && vals.length === 0) res.error ??= fail(cmd, `Thiếu ${label(p)}`);
    for (const v of vals) {
      const bad = checkValue(p, v);
      if (bad) res.error ??= fail(cmd, bad);
    }
  });
  const last = cmd.args[cmd.args.length - 1];
  if (res.positional.length > cmd.args.length && !last?.multiple) {
    res.error ??= fail(cmd, `Thừa tham số "${res.positional[cmd.args.length]}"`);
  }
  for (const f of cmd.flags) {
    if (f.required && !res.flags.some((u) => u.name === f.name)) res.error ??= fail(cmd, `Thiếu cờ --${f.name}`);
  }
  return res;
}

/** Mảng args gửi cho Rust (không gồm `--json`, Rust tự thêm). */
export function toArgs(p: ParsedLine): string[] {
  const out = [...p.cmd.name.split(" "), ...p.positional];
  for (const f of p.flags) {
    out.push(`--${f.name}`);
    if (f.value !== true) out.push(f.value);
  }
  return out;
}

/** Tên cờ chưa dùng khớp tiền tố `--xx`. */
export function suggestFlags(cmd: Command, used: string[], prefix: string): string[] {
  const q = prefix.replace(/^--/, "").toLowerCase();
  return cmd.flags.map((f) => f.name).filter((n) => !used.includes(n) && n.startsWith(q));
}
