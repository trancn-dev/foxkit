/** `Ctrl+Alt+K` từ phím vừa bấm; null nếu chưa đủ (chỉ có phím bổ trợ, hoặc không có phím bổ trợ nào). */
export function combo(e: Pick<KeyboardEvent, "key" | "ctrlKey" | "altKey" | "shiftKey" | "metaKey">): string | null {
  if (["Control", "Alt", "Shift", "Meta"].includes(e.key)) return null;
  const mods = [e.ctrlKey && "Ctrl", e.altKey && "Alt", e.shiftKey && "Shift", e.metaKey && "Super"].filter(Boolean) as string[];
  if (mods.length === 0) return null;
  const k = e.key === " " ? "Space" : e.key.length === 1 ? e.key.toUpperCase() : e.key;
  return [...mods, k].join("+");
}
