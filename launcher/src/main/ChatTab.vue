<script setup lang="ts">
import { invoke } from "@tauri-apps/api/core";
import { nextTick, onMounted, ref, watch } from "vue";
import Composer from "../components/Composer.vue";
import ResultView from "../components/ResultView.vue";
import type { Session } from "../lib/session";

const props = defineProps<{ session: Session }>();
const composer = ref<InstanceType<typeof Composer> | null>(null);
const list = ref<HTMLElement | null>(null);

// Mục mới nhất ở dưới, tự cuộn xuống khi có mục mới.
watch(
  () => props.session.items.value.length,
  () => nextTick(() => list.value && (list.value.scrollTop = list.value.scrollHeight)),
);

async function shrink() {
  await props.session.sendDraft("bar", composer.value?.getText() ?? "");
  composer.value?.setText("");
  await invoke("show_bar");
}

onMounted(async () => {
  await props.session.onDraft((t) => composer.value?.setText(t));
  composer.value?.focus();
});
defineExpose({ focus: () => composer.value?.focus() });
</script>

<template>
  <div class="chat">
    <div class="chat-head">
      <span class="dim">Chat lệnh — dùng chung với ô nhanh ({{ session.config.value.hotkey }})</span>
      <button class="btn small" @click="shrink">Thu nhỏ</button>
    </div>
    <div ref="list" class="thread full">
      <div v-if="!session.items.value.length" class="dim empty">Chưa có lệnh nào. Gõ <code>uuid</code> hoặc <code>port --list</code> ở ô bên dưới.</div>
      <div v-for="it in session.items.value" :key="it.id" class="entry">
        <div class="line" @click="composer?.setText(it.line)">{{ it.line }}</div>
        <ResultView :item="it" :session="session" wide />
      </div>
    </div>
    <Composer ref="composer" :session="session" />
  </div>
</template>
