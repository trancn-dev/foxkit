<script setup lang="ts">
// Ô lệnh nhanh (chat mini), cửa sổ `bar`.
import { invoke } from "@tauri-apps/api/core";
import { getCurrentWindow, LogicalSize } from "@tauri-apps/api/window";
import { nextTick, onMounted, ref, watch } from "vue";
import Composer from "./components/Composer.vue";
import ResultView from "./components/ResultView.vue";
import { Session } from "./lib/session";
import { listen } from "@tauri-apps/api/event";

const session = new Session("bar");
const composer = ref<InstanceType<typeof Composer> | null>(null);
const root = ref<HTMLElement | null>(null);

const hide = () => invoke("hide_window");
const expand = async () => {
  await session.sendDraft("main", composer.value?.getText() ?? "");
  composer.value?.setText("");
  await invoke("show_main", { tab: "chat" });
};

// Cao theo nội dung, tối đa 480 px (launcher.md 6.2).
async function fit() {
  await nextTick();
  const h = Math.min(480, Math.max(90, (root.value?.scrollHeight ?? 90) + 2));
  await getCurrentWindow().setSize(new LogicalSize(680, h));
}

onMounted(async () => {
  await session.init();
  // Rust phát `bar-shown` sau khi đã chụp ngữ cảnh và hiện cửa sổ.
  await listen("bar-shown", async () => {
    await session.reload().catch(() => undefined);
    await session.refreshContext();
    composer.value?.focus(true);
    await fit();
  });
  await listen("reload", () => session.reload());
  await session.onDraft((t) => composer.value?.setText(t));
  await fit();
});
watch([session.items, session.error, session.prompt], fit, { deep: true });
</script>

<template>
  <div ref="root" class="bar">
    <Composer ref="composer" :session="session" @hide="hide">
      <template #actions>
        <button class="btn small" title="Mở giao diện chính" @click="expand">Mở rộng</button>
      </template>
    </Composer>
    <div class="thread quick">
      <div v-for="it in session.quick.value" :key="it.id" class="entry">
        <div class="line" :title="'Bấm để điền lại'" @click="composer?.setText(it.line)">{{ it.line }}</div>
        <ResultView :item="it" :session="session" />
      </div>
    </div>
  </div>
</template>
