<script setup lang="ts">
import { invoke } from "@tauri-apps/api/core";
import { writeText } from "@tauri-apps/plugin-clipboard-manager";
import { onMounted, ref } from "vue";
import type { AppError, Session } from "../lib/session";

interface Info { path: string; files: number; commands: number; skipped: number; scanned_at: number; status: string }
interface Report { added: number; changed: number; removed: number; skipped_files: number; folders_missing: string[]; messages: string[] }

const props = defineProps<{ session: Session }>();
const folders = ref<Info[]>([]);
const report = ref<string>("");
const log = ref("");
const error = ref<AppError | null>(null);
const copied = ref(false);

const SAMPLE = JSON.stringify(
  { $schema: "https://.../foxkit-command.schema.json", schema: 1, commands: [{ name: "hello", title: "Xin chào", description: "Ví dụ lệnh ngoài", category: "life", args: [], flags: [], output: "text", exec: { program: "cmd", prefix: ["/c", "echo", "xin chao"], cwd: "file", json: false } }] },
  null,
  2,
);

async function refresh() {
  folders.value = await invoke<Info[]>("folders_info");
  log.value = await invoke<string>("scan_log");
}
async function after(line: string, r: Report) {
  report.value = `+${r.added} mới, ~${r.changed} đổi, −${r.removed} không còn, ${r.skipped_files} file bị bỏ` + (r.messages.length ? "\n" + r.messages.join("\n") : "");
  await props.session.reload();
  await refresh();
}
async function guard(f: () => Promise<void>) {
  error.value = null;
  try {
    await f();
  } catch (e) {
    error.value = e as AppError;
  }
}
const add = () => guard(async () => {
  const path = await props.session.pickFolder();
  if (path) await after("add", await invoke<Report>("add_folder", { path }));
});
const rescan = (path?: string) => guard(async () => after("scan", await invoke<Report>("scan", { folders: path ? [path] : null })));
const forget = (path: string) => guard(async () => {
  await invoke("forget_folder", { path });
  report.value = `Đã bỏ ${path}`;
  await props.session.reload();
  await refresh();
});
async function copySample() {
  await writeText(SAMPLE);
  copied.value = true;
  setTimeout(() => (copied.value = false), 1500);
}
const when = (s: number) => (s ? new Date(s * 1000).toLocaleString("vi-VN") : "chưa quét");

onMounted(refresh);
</script>

<template>
  <div class="pad folders">
    <div class="row">
      <button class="btn primary" @click="add">Thêm thư mục…</button>
      <button class="btn" :disabled="!folders.length" @click="rescan()">Quét lại tất cả</button>
      <span class="dim">Không tự quét: chỉ quét khi bạn bấm (hoặc gõ <code>:scan</code>).</span>
    </div>
    <div v-if="error" class="err">{{ error.message }} <span class="dim">{{ error.hint }}</span></div>
    <table v-if="folders.length" class="list">
      <thead><tr><th>Thư mục</th><th>File</th><th>Lệnh</th><th>Quét lần cuối</th><th>Trạng thái</th><th></th></tr></thead>
      <tbody>
        <tr v-for="f in folders" :key="f.path">
          <td class="mono">{{ f.path }}</td><td>{{ f.files }}<span v-if="f.skipped" class="err"> (+{{ f.skipped }} bỏ)</span></td><td>{{ f.commands }}</td>
          <td>{{ when(f.scanned_at) }}</td><td :class="f.status === 'missing' ? 'err' : ''">{{ f.status === "missing" ? "không tìm thấy" : f.status }}</td>
          <td class="actions"><button class="btn small" @click="rescan(f.path)">Quét lại</button> <button class="btn small" @click="forget(f.path)">Bỏ</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="dim">Chưa có thư mục lệnh nào.</div>
    <pre v-if="report" class="pre">{{ report }}</pre>

    <h3>Quy ước vị trí file</h3>
    <p class="dim">
      Chọn một thư mục gốc (vd <code>D:\TranCN\foxkit</code>). Launcher chỉ đọc file tên chính xác <code>foxkit.json</code> ở gốc và ở thư mục con trực tiếp (một cấp).
      Mỗi tool một <code>foxkit.json</code> nằm cạnh code; đường dẫn tương đối trong file tính từ thư mục chứa nó.
    </p>
    <button class="btn" @click="copySample">{{ copied ? "Đã copy" : "Copy mẫu foxkit.json" }}</button>

    <h3>scan.log</h3>
    <pre class="pre log">{{ log || "(chưa có)" }}</pre>
  </div>
</template>
