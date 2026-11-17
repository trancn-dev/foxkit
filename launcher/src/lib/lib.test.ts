import { describe, expect, it } from "vitest";
import sample from "./testdata/commands.json";
import { formToArgs } from "./form";
import { exampleFor, ghost } from "./ghost";
import { normalize, score, search } from "./fuzzy";
import { type Manifest, normalizeManifest } from "./manifest";
import { expand } from "./placeholders";
import { childrenOf, findCommand, parseLine, suggestFlags, syntax, toArgs } from "./resolve";
import { ThreadStore, quickView, trim, type ThreadBackend, type ThreadItem } from "./thread";
import { tokenize } from "./tokenize";

const m: Manifest = normalizeManifest(sample);
const cmds = m.commands;
const T = (s: string) => tokenize(s).tokens;
const noCtx = { dir: null, files: [] as string[], clip: null };

describe("tokenize", () => {
  it.each([
    ["port 8000", ["port", "8000"]],
    ['find "C:\\My Files" --ext pdf', ["find", "C:\\My Files", "--ext", "pdf"]],
    ["a    b", ["a", "b"]],
    ["foxkit port 8000", ["port", "8000"]],
    ['say "chưa đóng', ["say", "chưa đóng"]],
  ])("%s", (input, want) => expect(tokenize(input).tokens).toEqual(want));

  it("báo nháy chưa đóng và khoảng trắng cuối", () => {
    expect(tokenize('a "b').unclosed).toBe(true);
    expect(tokenize("a ").trailingSpace).toBe(true);
    expect(tokenize("a").trailingSpace).toBe(false);
  });
});

describe("fuzzy", () => {
  it("bỏ dấu", () => expect(normalize("Đổi tên")).toBe("doi ten"));
  it("prt khớp port", () => expect(search("prt", cmds).map((c) => c.name)).toContain("port"));
  it("doi ten khớp title Đổi tên hàng loạt", () => expect(search("doi ten", cmds)[0].name).toBe("rename"));
  it("khớp đầu name xếp trên chứa chuỗi con", () => {
    const port = cmds.find((c) => c.name === "port")!;
    const size = cmds.find((c) => c.name === "size")!;
    expect(score("por", port)).toBeGreaterThan(score("por", size));
  });
  it("1-2 ký tự chỉ lấy đầu name/đầu từ title", () => {
    const names = search("h", cmds).map((c) => c.name);
    expect(names).toEqual(expect.arrayContaining(["hash", "hosts add", "hosts rm"]));
    expect(names).not.toContain("uuid"); // chỉ có chữ h ở giữa description, không phải đầu từ
    expect(search("p", cmds).map((c) => c.name)).not.toContain("size");
  });
});

describe("resolve", () => {
  it("hosts add khớp lệnh con", () => {
    const p = parseLine(T("hosts add 1.2.3.4 a.test"), cmds)!;
    expect(p.cmd.name).toBe("hosts add");
    expect(p.error).toBeUndefined();
    expect(toArgs(p)).toEqual(["hosts", "add", "1.2.3.4", "a.test"]);
  });
  it("hosts (lệnh cha) không khớp, gợi ý lệnh con", () => {
    expect(findCommand(T("hosts"), cmds)).toBeNull();
    expect(childrenOf(T("hosts"), cmds).map((c) => c.name)).toEqual(["hosts add", "hosts rm"]);
  });
  it("--out=x tách đúng", () => {
    const p = parseLine(T("find D:\\x --ext=pdf"), cmds)!;
    expect(p.flags).toEqual([{ name: "ext", value: "pdf" }]);
  });
  it("port 70000 → ARG_INVALID", () => {
    expect(parseLine(T("port 70000"), cmds)!.error?.message).toBe("Port phải là số từ 1 đến 65535");
  });
  it("cờ không tồn tại / thiếu giá trị / arg thừa / thiếu required", () => {
    expect(parseLine(T("port --nope"), cmds)!.error?.message).toContain("--nope");
    expect(parseLine(T("find D:\\x --ext"), cmds)!.error?.message).toContain("cần một giá trị");
    expect(parseLine(T("port 80 90"), cmds)!.error?.message).toContain("Thừa");
    expect(parseLine(T("hosts add 1.2.3.4"), cmds)!.error?.message).toBe("Thiếu Host");
  });
  it("enum sai", () => expect(parseLine(T("hash a.txt --algo zz"), cmds)!.error).toBeDefined());
  it("multiple nhận nhiều arg; --json bị bỏ", () => {
    const p = parseLine(T("hash a b c --json"), cmds)!;
    expect(p.error).toBeUndefined();
    expect(toArgs(p)).toEqual(["hash", "a", "b", "c"]);
  });
  it("gợi ý cờ chưa dùng", () => {
    const port = cmds.find((c) => c.name === "port")!;
    expect(suggestFlags(port, ["kill"], "--")).toEqual(["list"]);
  });
  it("syntax", () => expect(syntax(cmds.find((c) => c.name === "port")!)).toBe("port [Port] [--kill] [--list]"));
});

describe("placeholders", () => {
  const ctx = { enabled: true, dir: "C:\\My Files", files: ["a.txt", "b.txt"], clip: "8000" };
  it("thay đúng", () => {
    expect(expand(["size", "@dir"], ctx)).toEqual(["size", "C:\\My Files"]);
    expect(expand(["hash", "@files"], ctx)).toEqual(["hash", "a.txt", "b.txt"]);
    expect(expand(["hash", "@file"], ctx)).toEqual(["hash", "a.txt"]);
    expect(expand(["port", "@clip"], ctx)).toEqual(["port", "8000"]);
  });
  it("@@dir thành @dir; email giữ nguyên", () => {
    expect(expand(["@@dir", "a@b.com"], ctx)).toEqual(["@dir", "a@b.com"]);
  });
  it("thiếu giá trị / context:false → CONTEXT_MISSING", () => {
    expect(() => expand(["@dir"], { ...ctx, dir: null })).toThrowError(expect.objectContaining({ code: "CONTEXT_MISSING" }));
    expect(() => expand(["@clip"], { ...ctx, enabled: false })).toThrowError(expect.objectContaining({ code: "CONTEXT_MISSING" }));
    expect(() => expand(["@files"], { ...ctx, files: [] })).toThrowError(expect.objectContaining({ code: "CONTEXT_MISSING" }));
  });
});

describe("ghost", () => {
  it("po → rt", () => expect(ghost("po", cmds, noCtx)).toEqual({ text: "rt", acceptable: true }));
  it("hosts a → dd", () => expect(ghost("hosts a", cmds, noCtx)?.text).toBe("dd"));
  it("port + clipboard 8000 → 8000", () => expect(ghost("port ", cmds, { ...noCtx, clip: "8000" })?.text).toBe("8000"));
  it("clipboard ngoài khoảng hoặc không phải số → không ghost số", () => {
    expect(ghost("port ", cmds, { ...noCtx, clip: "70000" })?.text).not.toBe("70000");
    expect(ghost("port ", cmds, { ...noCtx, clip: "abc" })?.text).not.toBe("abc");
  });
  it("size + dir có dấu cách → có nháy", () => {
    expect(ghost("size ", cmds, { ...noCtx, dir: "C:\\My Files" })?.text).toBe('"C:\\My Files"');
  });
  it("hash + file chọn → file đầu", () => expect(ghost("hash ", cmds, { ...noCtx, files: ["a.txt", "b.txt"] })?.text).toBe("a.txt"));
  it("đủ arg → cờ chưa dùng", () => {
    expect(ghost("port 8000 ", cmds, noCtx)?.text).toBe("--kill");
    expect(ghost("port 8000 --kill ", cmds, noCtx)?.text).toBe("--list");
  });
  it("đang gõ dở cờ", () => expect(ghost("port 8000 --ki", cmds, noCtx)?.text).toBe("ll"));
  it("arg bắt buộc chưa gõ → nhắc, không nhận bằng Tab", () => {
    expect(ghost("find ", cmds, noCtx)).toEqual({ text: "<Thư mục>", acceptable: false });
  });
  it("enum", () => {
    expect(ghost("hash a.txt --algo ", cmds, noCtx)?.text).toBe("md5");
    expect(ghost("hash a.txt --algo sha", cmds, noCtx)?.text).toBe("1");
  });
  it("string tự do → không đoán", () => expect(ghost("hosts add ", cmds, noCtx)?.acceptable).toBe(false));
  it("đã gõ đủ tên lệnh, chưa cách → không ghost", () => expect(ghost("port", cmds, noCtx)).toBeNull());
});

describe("exampleFor", () => {
  it("ô trống → random một ví dụ", () => expect(exampleFor("", cmds, () => 0)?.cmd).toBe("port 8000 --kill"));
  it("đã khớp lệnh → ví dụ đầu; gõ đúng ví dụ → ví dụ kế", () => {
    expect(exampleFor("port", cmds)?.cmd).toBe("port 8000 --kill");
    expect(exampleFor("port 8000 --kill", cmds)?.cmd).toBe("port --list");
  });
  it("lệnh không có examples → null, không bịa", () => expect(exampleFor("hash", cmds)).toBeNull());
  it("chưa khớp lệnh → ví dụ của gợi ý đầu có ví dụ", () => expect(exampleFor("uu", cmds)?.cmd).toBe("uuid --v 7"));
});

describe("form", () => {
  const port = cmds.find((c) => c.name === "port")!;
  it("giống gõ tay", () => {
    expect(formToArgs(port, { args: { port: "8000" }, flags: { kill: true } }).args).toEqual(["port", "8000", "--kill"]);
    expect(formToArgs(port, { args: { port: "8000" }, flags: { kill: false } }).args).toEqual(["port", "8000"]);
  });
  it("giá trị trống không tạo cờ; multiple nhiều dòng", () => {
    const hash = cmds.find((c) => c.name === "hash")!;
    expect(formToArgs(hash, { args: { file: ["a b.txt", " ", "c.txt"] }, flags: { algo: "" } }).args).toEqual(["hash", "a b.txt", "c.txt"]);
  });
  it("cùng thông báo lỗi với resolve", () => {
    expect(formToArgs(port, { args: { port: "70000" }, flags: {} }).error?.message).toBe("Port phải là số từ 1 đến 65535");
    const hosts = cmds.find((c) => c.name === "hosts add")!;
    expect(formToArgs(hosts, { args: { ip: "1.1.1.1" }, flags: {} }).error?.message).toBe("Thiếu Host");
  });
});

describe("thread", () => {
  const item = (i: number): ThreadItem => ({ id: i, line: `c${i}`, status: "done", ts: i });
  it("quá 100 mục bỏ cái cũ nhất", () => {
    const t = trim(Array.from({ length: 105 }, (_, i) => item(i + 1)));
    expect(t).toHaveLength(100);
    expect(t[0].id).toBe(6);
  });
  it("ô nhanh lấy 20 mới nhất, mới ở trên", () => {
    const q = quickView(Array.from({ length: 30 }, (_, i) => item(i + 1)));
    expect(q).toHaveLength(20);
    expect(q[0].id).toBe(30);
  });
  it("store: đang chạy → kết quả, nhận sự kiện, clear", async () => {
    let items: ThreadItem[] = [];
    const subs: (() => void)[] = [];
    const be: ThreadBackend = {
      list: async () => items.slice(),
      add: async (it) => { items.push({ ...it, id: items.length + 1 }); subs.forEach((f) => f()); return items.length; },
      update: async (id, it) => { items = items.map((x) => (x.id === id ? it : x)); subs.forEach((f) => f()); },
      remove: async (id) => { items = items.filter((x) => x.id !== id); subs.forEach((f) => f()); },
      clear: async () => { items = []; subs.forEach((f) => f()); },
      onChanged: async (cb) => { subs.push(cb); return () => {}; },
    };
    const s = new ThreadStore(be);
    await s.start();
    const id = await s.begin("uuid");
    await new Promise((r) => setTimeout(r, 0));
    expect(s.items[0].status).toBe("running");
    await s.finish(id, "uuid", "done", { ok: true });
    await new Promise((r) => setTimeout(r, 0));
    expect(s.items[0]).toMatchObject({ status: "done", result: { ok: true } });
    const id2 = await s.begin("pass");
    await s.remove(id);
    await new Promise((r) => setTimeout(r, 0));
    expect(s.items.map((x) => x.id)).toEqual([id2]); // xóa một mục, mục kia còn
    await s.clear();
    await new Promise((r) => setTimeout(r, 0));
    expect(s.items).toEqual([]);
  });
});

describe("manifest", () => {
  it("schema khác 1 → MANIFEST_UNSUPPORTED", () => {
    expect(() => normalizeManifest({ schema: 2, commands: [] })).toThrowError(expect.objectContaining({ code: "MANIFEST_UNSUPPORTED" }));
  });
  it("không phải manifest → MANIFEST_FAILED", () => {
    expect(() => normalizeManifest(null)).toThrowError(expect.objectContaining({ code: "MANIFEST_FAILED" }));
  });
});

describe("appcmds", () => {
  it("mặc định (chưa cấu hình) không gợi ý gì; cấu hình theo tên exe, không phân biệt hoa thường", async () => {
    const { preferredFor, exeName } = await import("./appcmds");
    const chrome = String.raw`C:\Program Files\Google\Chrome\Application\chrome.exe`;
    expect(preferredFor(chrome)).toEqual([]);
    const hints = [{ app: "Chrome", commands: ["http", "dns"] }, { app: "msedge.exe", commands: ["jwt"] }];
    expect(preferredFor(chrome, hints)).toEqual(["http", "dns"]);
    expect(preferredFor(String.raw`C:\x\MSEDGE.EXE`, hints)).toEqual(["jwt"]);
    expect(preferredFor(String.raw`C:\Windows\notepad.exe`, hints)).toEqual([]);
    expect(preferredFor("", hints)).toEqual([]);
    expect(exeName(String.raw`C:\a\Code.exe`)).toBe("code.exe");
    expect(exeName("")).toBe("");
  });
});

describe("exampleFor theo app", () => {
  it("ô trống + app có bộ lệnh riêng → ví dụ chỉ từ bộ đó", () => {
    const withEx = (name: string, cmd: string) => ({ ...cmds.find((c) => c.name === "port")!, name, examples: [{ cmd, desc: "" }] });
    const list = [withEx("http", "http example.com"), withEx("zip", "zip pack @dir")];
    expect(exampleFor("", list, () => 0, ["http", "dns"])?.cmd).toBe("http example.com");
    expect(exampleFor("", list, () => 1, [])?.cmd).toBe("zip pack @dir");
  });
});
