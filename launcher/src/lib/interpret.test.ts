import { describe, expect, it } from "vitest";
import sample from "./testdata/commands.json";
import { copyForResult, copyText, interpret, type Result } from "./interpret";
import { normalizeManifest, type Command } from "./manifest";

const cmds = normalizeManifest(sample).commands;
const port = cmds.find((c) => c.name === "port")!;
const ext = (json: boolean): Command => ({ ...port, name: "hello", source: "external", exec: { json } } as Command);
const ok = (data: unknown, output: "kv" | "table" | "text" = "kv"): Extract<Result, { kind: "ok" }> => ({ kind: "ok", output, data, raw: "" });

describe("interpret", () => {
  it("exit 0 + envelope → ok theo output của lệnh", () => {
    const r = interpret({ exit_code: 0, stdout: '{"ok":true,"data":{"port":8000}}', stderr: "" }, port);
    expect(r).toMatchObject({ kind: "ok", output: "kv", data: { port: 8000 } });
  });
  it("exit 2 → lỗi từ stderr, không có nút admin", () => {
    const r = interpret({ exit_code: 2, stdout: "", stderr: '{"ok":false,"error":{"code":"PORT_INVALID","message":"Port sai","hint":"Ví dụ"}}' }, port);
    expect(r).toMatchObject({ kind: "error", code: "PORT_INVALID", message: "Port sai", hint: "Ví dụ", canAdmin: false });
  });
  it("exit 3 → canAdmin (trừ lệnh chạy lâu)", () => {
    const err = '{"ok":false,"error":{"code":"NEED_ADMIN","message":"Cần admin"}}';
    expect(interpret({ exit_code: 3, stdout: "", stderr: err }, port, ["port", "80"])).toMatchObject({ canAdmin: true, args: ["port", "80"] });
    expect(interpret({ exit_code: 3, stdout: "", stderr: err }, { ...port, longRunning: true })).toMatchObject({ canAdmin: false });
  });
  it("không phải JSON → BAD_OUTPUT kèm nguyên văn", () => {
    const r = interpret({ exit_code: 0, stdout: "xin chao", stderr: "" }, port);
    expect(r).toMatchObject({ kind: "error", code: "BAD_OUTPUT", detail: "xin chao" });
    expect(interpret({ exit_code: 1, stdout: "", stderr: "boom" }, port)).toMatchObject({ code: "BAD_OUTPUT" });
  });
  it("lệnh ngoài json:false → văn bản; lỗi → EXTERNAL_FAILED", () => {
    expect(interpret({ exit_code: 0, stdout: "xin chao\r\n", stderr: "" }, ext(false))).toMatchObject({ kind: "ok", output: "text", data: "xin chao" });
    expect(interpret({ exit_code: 5, stdout: "", stderr: "hỏng" }, ext(false))).toMatchObject({ code: "EXTERNAL_FAILED", message: "Lệnh hello kết thúc với mã 5", detail: "hỏng" });
  });
  it("lệnh ngoài json:true dùng envelope như lệnh thường", () => {
    expect(interpret({ exit_code: 0, stdout: '{"ok":true,"data":"x"}', stderr: "" }, ext(true))).toMatchObject({ kind: "ok", data: "x" });
  });
});

describe("copy", () => {
  it("string → chính nó", () => expect(copyText(ok("abc", "text"))).toBe("abc"));
  it("kv một khóa → giá trị", () => expect(copyText(ok({ result: "aGVsbG8=" }))).toBe("aGVsbG8="));
  it("table → TSV", () => {
    const t = copyText(ok({ items: [{ pid: 1, name: "a b" }, { pid: 2, name: "c\td" }] }, "table"));
    expect(t).toBe("pid\tname\n1\ta b\n2\tc d");
  });
  it("còn lại → JSON định dạng", () => expect(copyText(ok({ a: 1, b: 2 }))).toBe('{\n  "a": 1,\n  "b": 2\n}'));
  it("danh sách chuỗi (uuid, pass) → mỗi dòng một giá trị", () => {
    expect(copyForResult(ok({ version: 4, items: ["u1", "u2"] }, "text"))).toBe("u1\nu2");
  });
  it("lỗi/stream không copy", () => expect(copyForResult({ kind: "stream", lines: [], running: false })).toBeNull());
});
