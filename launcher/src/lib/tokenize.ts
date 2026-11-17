export interface Tokens {
  tokens: string[];
  /** Nháy kép mở mà chưa đóng (coi như đóng ở cuối). */
  unclosed: boolean;
  /** Ô nhập kết thúc bằng khoảng trắng (đang bắt đầu token mới). */
  trailingSpace: boolean;
}

/** Tách theo khoảng trắng; nháy kép gom thành một token; `\` giữ nguyên (đường dẫn Windows). */
export function tokenize(input: string): Tokens {
  const tokens: string[] = [];
  let cur = "";
  let has = false;
  let inQuote = false;
  for (const ch of input) {
    if (ch === '"') {
      inQuote = !inQuote;
      has = true;
    } else if (!inQuote && /\s/.test(ch)) {
      if (has) tokens.push(cur);
      cur = "";
      has = false;
    } else {
      cur += ch;
      has = true;
    }
  }
  if (has) tokens.push(cur);
  // Người dùng quen gõ `foxkit port 8000` như ở terminal: bỏ token đầu.
  if (tokens[0]?.toLowerCase() === "foxkit") tokens.shift();
  return { tokens, unclosed: inQuote, trailingSpace: !inQuote && /\s$/.test(input) };
}

/** Đưa một token về dạng gõ được (có nháy kép nếu chứa khoảng trắng). */
export function quoteToken(t: string): string {
  return /\s/.test(t) ? `"${t}"` : t;
}
