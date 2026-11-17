<script setup lang="ts">
import { invoke } from "@tauri-apps/api/core";
import { open } from "@tauri-apps/plugin-dialog";
import { computed, onMounted, ref, watch } from "vue";
import { exeName, type AppHint } from "../lib/appcmds";
import type { AppError, Session } from "../lib/session";

const props = defineProps<{ session: Session }>();
const rows = ref<AppHint[]>([]);
const adding = ref<string[]>([]); // ô "thêm lệnh" của từng dòng
const error = ref<AppError | null>(null);
const saved = ref(false);

// Tên lệnh để gợi ý trong ô thêm (bỏ lệnh nội bộ bắt đầu bằng ":").
const names = computed(() => props.session.commands.value.map((c) => c.name).filter((n) => !n.startsWith(":")));

function load() {
  rows.value = props.session.config.value.appHints.map((h) => ({ app: h.app, commands: [...h.commands] }));
  adding.value = rows.value.map(() => "");
}
onMounted(load);
// cấu hình có thể tải xong sau khi tab mở (hoặc được nạp lại từ tray)
watch(() => props.session.config.value.appHints, load);

function addRow() {
  rows.value.push({ app: "", commands: [] });
  adding.value.push("");
}
function removeRow(i: number) {
  rows.value.splice(i, 1);
  adding.value.splice(i, 1);
}
async function pickExe(i: number) {
  const p = await open({ multiple: false, filters: [{ name: "Ứng dụng", extensions: ["exe"] }] });
  if (typeof p === "string") rows.value[i].app = exeName(p);
}
function addCommand(i: number) {
  const n = adding.value[i].trim();
  if (n && names.value.includes(n) && !rows.value[i].commands.includes(n)) rows.value[i].commands.push(n);
  adding.value[i] = "";
}
function removeCommand(i: number, n: string) {
  rows.value[i].commands = rows.value[i].commands.filter((c) => c !== n);
}

async function save() {
  error.value = null;
  saved.value = false;
  // bỏ dòng trống, chuẩn hóa tên exe
  const appHints = rows.value.map((r) => ({ app: exeName(r.app), commands: r.commands })).filter((r) => r.app && r.commands.length);
  try {
    await invoke("save_config", { cfg: { ...props.session.config.value, appHints } });
    await props.session.reload();
    load();
    saved.value = true;
  } catch (e) {
    error.value = e as AppError;
  }
}
</script>

<template>
  <div class="pad apps">
    <p class="dim">
      Chọn lệnh hiện sẵn khi mở ô nhanh lúc một app đang focus. Mặc định không gợi ý gì. Tên app là tên file exe, ví dụ <code>chrome.exe</code>, <code>Code.exe</code>.
    </p>
    <div v-if="!rows.length" class="dim">Chưa có app nào.</div>
    <div v-for="(r, i) in rows" :key="i" class="approw">
      <div class="inline">
        <input v-model="r.app" spellcheck="false" placeholder="chrome.exe" />
        <button class="btn small" @click="pickExe(i)">Chọn .exe…</button>
        <button class="btn small" title="Bỏ app này" @click="removeRow(i)">Xóa</button>
      </div>
      <div class="chips">
        <span v-for="c in r.commands" :key="c" class="chip">{{ c }} <a href="#" title="Bỏ lệnh" @click.prevent="removeCommand(i, c)">×</a></span>
        <input v-model="adding[i]" :list="'cmds-' + i" placeholder="thêm lệnh…" spellcheck="false" @change="addCommand(i)" @keydown.enter.prevent="addCommand(i)" />
        <datalist :id="'cmds-' + i"><option v-for="n in names" :key="n" :value="n" /></datalist>
      </div>
    </div>
    <div v-if="error" class="err">{{ error.message }} <span class="dim">{{ error.hint }}</span></div>
    <div class="row">
      <button class="btn" @click="addRow">Thêm app</button>
      <button class="btn primary" @click="save">Lưu</button>
      <span v-if="saved" class="ok">Đã lưu và áp dụng.</span>
    </div>
  </div>
</template>

<style scoped>
.approw { margin: 10px 0; padding: 8px 10px; border: 1px solid var(--line); border-radius: 8px; }
.inline { display: flex; gap: 6px; margin-bottom: 6px; }
.inline input { flex: 1; }
.chips { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.chips .chip { display: inline-flex; gap: 6px; max-width: none; }
.chips .chip a { color: var(--dim); text-decoration: none; }
.chips input { width: 140px; }
</style>
