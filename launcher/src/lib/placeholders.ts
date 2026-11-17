export interface PlaceholderCtx {
  /** false khi config `context: false`: mọi placeholder báo CONTEXT_MISSING. */
  enabled: boolean;
  dir?: string | null;
  files: string[];
  clip?: string | null;
}

export interface ContextError {
  code: "CONTEXT_MISSING";
  message: string;
  hint: string;
}

const HINT = "Bấm phím tắt khi đang ở cửa sổ Explorer hoặc VS Code đã mở thư mục, hoặc gõ đường dẫn trực tiếp";

function missing(what: string, hint = HINT): ContextError {
  return { code: "CONTEXT_MISSING", message: `Không xác định được ${what}`, hint };
}

/**
 * Mở rộng `@dir @file @files @clip` (cả token) trước khi chạy. `@@dir` → `@dir` nguyên văn.
 * Token `@` khác (vd email) giữ nguyên. Ném ContextError nếu thiếu giá trị.
 */
export function expand(tokens: string[], ctx: PlaceholderCtx): string[] {
  const out: string[] = [];
  for (const t of tokens) {
    switch (t) {
      case "@dir":
        if (!ctx.enabled || !ctx.dir) throw missing("thư mục hiện tại");
        out.push(ctx.dir);
        break;
      case "@file":
        if (!ctx.enabled || !ctx.files[0]) throw missing("file đang chọn");
        out.push(ctx.files[0]);
        break;
      case "@files":
        if (!ctx.enabled || ctx.files.length === 0) throw missing("file đang chọn");
        out.push(...ctx.files);
        break;
      case "@clip":
        if (!ctx.enabled || !ctx.clip) throw missing("nội dung clipboard");
        out.push(ctx.clip);
        break;
      default:
        out.push(/^@@(dir|file|files|clip)$/.test(t) ? t.slice(1) : t);
    }
  }
  return out;
}
