<script setup lang="ts">
import { invoke } from "@tauri-apps/api/core";
import { open } from "@tauri-apps/plugin-dialog";
import { onMounted, ref } from "vue";
import { combo } from "../lib/hotkey";
import type { AppConfig, AppError, Session } from "../lib/session";

const props = defineProps<{ session: Session }>();
const cfg = ref<AppConfig>({ ...props.session.config.value });
const error = ref<AppError | null>(null);
const saved = ref(false);
const versionText = ref("");
const warn = ref("");

function onHotkey(e: KeyboardEvent) {
  e.preventDefault();
  const c = combo(e);
  if (!c) return;
  cfg.value.hotkey = c;
  warn.value = c.includes("Super") ? "Tổ hợp có phím Win thường bị Windows giữ." : "";
}

async function pickExe() {
  const p = await open({ multiple: false, filters: [{ name: "foxkit", extensions: ["exe"] }] });
  if (typeof p === "string") cfg.value.foxkitPath = p;
}

async function check() {
  versionText.value = "";
  try {
    versionText.value = await invoke<string>("check_foxkit", { path: cfg.value.foxkitPath });
  } catch (e) {
    versionText.value = (e as AppError).message;
  }
}

async function save() {
  error.value = null;
  saved.value = false;
  try {
    await invoke("save_config", { cfg: cfg.value });
    await props.session.reload();
    cfg.value = { ...props.session.config.value };
    saved.value = true;
  } catch (e) {
    error.value = e as AppError;
  }
}

onMounted(() => (cfg.value = { ...props.session.config.value }));
</script>

<template>
  <div class="pad settings">
    <div class="field-row">
      <label>Phím tắt mở ô lệnh</label>
      <input :value="cfg.hotkey" readonly placeholder="Bấm vào đây rồi nhấn tổ hợp phím" @keydown="onHotkey" />
      <div v-if="warn" class="dim small">{{ warn }}</div>
    </div>
    <label class="check"><input v-model="cfg.autostart" type="checkbox" /> Khởi động cùng Windows (chạy ngầm ở tray)</label>
    <label class="check"><input v-model="cfg.autoCopy" type="checkbox" /> Tự copy kết quả sau mỗi lệnh thành công</label>
    <label class="check"><input v-model="cfg.context" type="checkbox" /> Nhận diện app đang focus, thư mục và clipboard (ngữ cảnh)</label>
    <div class="field-row">
      <label>Đường dẫn foxkit.exe (để trống = tự tìm)</label>
      <div class="inline">
        <input v-model="cfg.foxkitPath" spellcheck="false" />
        <button class="btn small" @click="pickExe">Chọn…</button>
        <button class="btn small" @click="check">Kiểm tra</button>
      </div>
      <div v-if="versionText" class="dim small mono">{{ versionText }}</div>
    </div>
    <div v-if="error" class="err">{{ error.message }} <span class="dim">{{ error.hint }}</span></div>
    <div class="row">
      <button class="btn primary" @click="save">Lưu</button>
      <span v-if="saved" class="ok">Đã lưu và áp dụng.</span>
    </div>
  </div>
</template>
