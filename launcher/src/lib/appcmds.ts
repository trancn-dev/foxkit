/** Gợi ý riêng cho một app: tên exe (vd. `chrome.exe`) → các lệnh hiện sẵn khi ô nhanh mở lúc app đó đang focus. */
export interface AppHint {
  app: string;
  commands: string[];
}

/** So tên file exe, không phân biệt hoa thường, cho phép gõ thiếu `.exe`. */
export function exeName(s: string): string {
  const file = (s.split(/[\\/]/).pop() ?? "").trim().toLowerCase();
  return file.endsWith(".exe") ? file : file ? `${file}.exe` : "";
}

/** Lệnh nên gợi ý sẵn cho app đang focus (mặc định rỗng: chưa cấu hình thì không gợi ý gì). */
export function preferredFor(exe: string, hints: AppHint[] = []): string[] {
  const me = exeName(exe);
  if (!me) return [];
  return hints.filter((h) => exeName(h.app) === me).flatMap((h) => h.commands);
}
