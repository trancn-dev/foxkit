// Luồng chat dùng chung. Dữ liệu nằm trong state của Rust (chỉ trong bộ nhớ), file này là phần xem/đồng bộ ở frontend.

export interface ThreadItem {
  /** Rust gán khi thêm; 0 trước khi có id. */
  id: number;
  /** Dòng lệnh đã gõ. */
  line: string;
  status: "running" | "done" | "error";
  /** Kết quả đã chuẩn hóa để ResultView hiển thị (hoặc lỗi). Rust coi là JSON tùy ý. */
  result?: unknown;
  ts: number;
}

export const THREAD_MAX = 100;
export const QUICK_MAX = 20;

/** Ô nhanh: 20 mục mới nhất, mới nhất ở trên. */
export function quickView(items: ThreadItem[]): ThreadItem[] {
  return items.slice(-QUICK_MAX).reverse();
}

/** Giữ tối đa THREAD_MAX mục mới nhất. */
export function trim(items: ThreadItem[]): ThreadItem[] {
  return items.length > THREAD_MAX ? items.slice(items.length - THREAD_MAX) : items;
}

/** Phần "đường ống" tới Rust, tách ra để test bằng bản giả. */
export interface ThreadBackend {
  list(): Promise<ThreadItem[]>;
  add(item: ThreadItem): Promise<number>;
  update(id: number, item: ThreadItem): Promise<void>;
  clear(): Promise<void>;
  /** Đăng ký nhận sự kiện `thread-changed`; trả hàm hủy. */
  onChanged(cb: () => void): Promise<() => void>;
}

export class ThreadStore {
  items: ThreadItem[] = [];
  private listeners = new Set<(items: ThreadItem[]) => void>();
  private unlisten?: () => void;

  constructor(private backend: ThreadBackend) {}

  async start(): Promise<void> {
    await this.refresh();
    this.unlisten = await this.backend.onChanged(() => void this.refresh());
  }

  stop(): void {
    this.unlisten?.();
  }

  subscribe(cb: (items: ThreadItem[]) => void): () => void {
    this.listeners.add(cb);
    return () => this.listeners.delete(cb);
  }

  async refresh(): Promise<void> {
    this.items = trim(await this.backend.list());
    this.listeners.forEach((cb) => cb(this.items));
  }

  /** Thêm mục "đang chạy" và trả id để cập nhật sau. */
  async begin(line: string): Promise<number> {
    return this.backend.add({ id: 0, line, status: "running", ts: Date.now() });
  }

  async finish(id: number, line: string, status: "done" | "error", result: unknown): Promise<void> {
    await this.backend.update(id, { id, line, status, result, ts: Date.now() });
  }

  async clear(): Promise<void> {
    await this.backend.clear();
  }
}
