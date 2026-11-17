import { search } from "./fuzzy";
import { label, type Command, type Example, type Param } from "./manifest";
import { findCommand } from "./resolve";
import { quoteToken, tokenize } from "./tokenize";

export interface GhostCtx {
  dir?: string | null;
  files: string[];
  clip?: string | null;
}

export interface Ghost {
  text: string;
  /** false: chỉ nhắc (vd `<Port>`), không nhận bằng Tab. */
  acceptable: boolean;
}

const ok = (text: string): Ghost => ({ text, acceptable: true });

function enumRest(p: Param | undefined, typed: string): Ghost | null {
  if (p?.type !== "enum" || !p.options) return null;
  const o = p.options.find((x) => x.toLowerCase().startsWith(typed.toLowerCase()) && x.length > typed.length);
  return o ? ok(o.slice(typed.length)) : null;
}

/** Giá trị gợi ý (từ ngữ cảnh/clipboard/options) cho arg kế tiếp, hoặc null. */
function valueFor(p: Param, ctx: GhostCtx): string | null {
  switch (p.type) {
    case "dir":
      return ctx.dir || null;
    case "file":
      return ctx.files[0] || (p.allowDir && ctx.dir) || null;
    case "int": {
      const c = ctx.clip?.trim();
      if (c && /^-?\d+$/.test(c) && (p.min === undefined || +c >= p.min) && (p.max === undefined || +c <= p.max)) return c;
      return null;
    }
    case "enum":
      return p.options?.[0] ?? null;
    default:
      return null;
  }
}

/** Ghost text (phần gõ tiếp theo cú pháp), tính theo bảng ở launcher.md 6.4 bước 6. Chỉ dùng khi con trỏ ở cuối ô. */
export function ghost(input: string, commands: Command[], ctx: GhostCtx): Ghost | null {
  const { tokens, trailingSpace } = tokenize(input);
  if (tokens.length === 0) return null;
  const found = findCommand(tokens, commands);

  // 1. còn đang gõ tên lệnh
  if (!found) {
    const typed = tokens.join(" ").toLowerCase() + (trailingSpace ? " " : "");
    const cand = search(tokens.join(" "), commands, 20).find((c) => c.name.startsWith(typed) && c.name.length > typed.length);
    return cand ? ok(cand.name.slice(typed.length)) : null;
  }
  if (found.used === tokens.length && !trailingSpace) return null;

  const { cmd } = found;
  const rest = tokens.slice(found.used);
  const typing = !trailingSpace && rest.length > 0;
  const done = typing ? rest.slice(0, -1) : rest;

  // đếm positional đã đủ, cờ đã dùng, cờ đang chờ giá trị
  let positional = 0;
  const used: string[] = [];
  let pending: Param | null = null;
  for (let i = 0; i < done.length; i++) {
    const t = done[i];
    if (!t.startsWith("--")) {
      positional++;
      continue;
    }
    const name = t.slice(2).split("=")[0];
    used.push(name);
    const def = cmd.flags.find((f) => f.name === name);
    if (def && def.type !== "bool" && !t.includes("=")) {
      if (i + 1 < done.length && !done[i + 1].startsWith("--")) i++;
      else if (i === done.length - 1) pending = def;
    }
  }

  const argAt = (n: number): Param | undefined => cmd.args[n] ?? (cmd.args[cmd.args.length - 1]?.multiple ? cmd.args[cmd.args.length - 1] : undefined);

  if (typing) {
    const last = rest[rest.length - 1];
    if (last.startsWith("--")) {
      if (last.includes("=")) return null;
      const prefix = last.slice(2);
      const f = cmd.flags.find((x) => !used.includes(x.name) && x.name.startsWith(prefix) && x.name.length > prefix.length);
      return f ? ok(f.name.slice(prefix.length)) : null;
    }
    return enumRest(pending ?? argAt(positional) ?? undefined, last);
  }

  if (pending) {
    const v = pending.type === "enum" ? pending.options?.[0] : null;
    return v ? ok(v) : null;
  }
  const arg = argAt(positional);
  if (arg) {
    const v = valueFor(arg, ctx);
    if (v) return ok(quoteToken(v));
    if (arg.required) return { text: `<${label(arg)}>`, acceptable: false };
  }
  const requiredLeft = cmd.args.filter((a) => a.required).length - positional;
  if (requiredLeft > 0) return null;
  if (!trailingSpace) return null;
  const f = cmd.flags.find((x) => !used.includes(x.name));
  return f ? ok(`--${f.name}`) : null;
}

/** Ví dụ mờ phía trên ô nhập (launcher.md 6.4 bước 7). `pick` để test được tính ngẫu nhiên. */
export function exampleFor(input: string, commands: Command[], pick: (n: number) => number = (n) => Math.floor(Math.random() * n)): Example | null {
  const { tokens } = tokenize(input);
  if (tokens.length === 0) {
    const all = commands.flatMap((c) => c.examples ?? []);
    return all.length ? all[pick(all.length)] : null;
  }
  const found = findCommand(tokens, commands);
  if (!found) {
    const first = search(tokens.join(" "), commands, 8).find((c) => c.examples?.length);
    return first?.examples?.[0] ?? null;
  }
  const exs = found.cmd.examples ?? [];
  if (exs.length === 0) return null;
  const cur = tokens.join(" ");
  const i = exs.findIndex((e) => tokenize(e.cmd).tokens.join(" ") === cur);
  if (i < 0) return exs[0];
  return exs.length > 1 ? exs[(i + 1) % exs.length] : null;
}
