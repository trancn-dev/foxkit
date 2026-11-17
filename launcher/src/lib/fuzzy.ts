import type { Command } from "./manifest";

/** Chữ thường, bỏ dấu tiếng Việt (đ → d). */
export function normalize(s: string): string {
  return s.normalize("NFD").replace(/[̀-ͯ]/g, "").replace(/đ/g, "d").replace(/Đ/g, "d").toLowerCase();
}

function wordStarts(text: string, q: string): boolean {
  return text.split(/[^a-z0-9]+/).some((w) => w.startsWith(q));
}

function subsequence(text: string, q: string): boolean {
  let i = 0;
  for (const ch of text) if (ch === q[i] && ++i === q.length) return true;
  return q.length === 0;
}

/** Điểm khớp của truy vấn với một lệnh; 0 = không khớp. */
export function score(query: string, c: Command): number {
  const q = normalize(query.trim());
  if (!q) return 1;
  const name = normalize(c.name);
  const title = normalize(c.title);
  const desc = normalize(c.description);
  if (q.length <= 2) {
    if (name.startsWith(q)) return 100;
    return wordStarts(title, q) ? 60 : 0;
  }
  if (name.startsWith(q)) return 100;
  if (wordStarts(name, q) || wordStarts(title, q)) return 70;
  if (name.includes(q)) return 50;
  if (title.includes(q) || desc.includes(q)) return 40;
  if (subsequence(name, q) || subsequence(title, q)) return 20;
  return 0;
}

/** Tối đa `limit` lệnh khớp, điểm cao trước; cùng điểm giữ thứ tự manifest. */
export function search(query: string, commands: Command[], limit = 8): Command[] {
  return commands
    .map((c, i) => ({ c, s: score(query, c), i }))
    .filter((x) => x.s > 0)
    .sort((a, b) => b.s - a.s || a.i - b.i)
    .slice(0, limit)
    .map((x) => x.c);
}
