<script setup lang="ts">
// Ô lệnh nhanh (chat mini), cửa sổ `bar`.
import { invoke } from "@tauri-apps/api/core";
import { getCurrentWindow, LogicalSize } from "@tauri-apps/api/window";
import { computed, nextTick, onMounted, ref, watch } from "vue";
import Composer from "./components/Composer.vue";
import ResultView from "./components/ResultView.vue";
import { Session } from "./lib/session";
import { listen } from "@tauri-apps/api/event";

const session = new Session("bar");
const composer = ref<InstanceType<typeof Composer> | null>(null);
const inner = ref<HTMLElement | null>(null);
const capped = ref(false); // nội dung cao hơn 480px thì mới cho cuộn
// Kết quả cũ (từ lần mở trước) không hiện lại: ô nhanh luôn mở ở dạng gọn, chỉ có ô nhập.
const seenUpTo = ref(0);
const visible = computed(() => session.quick.value.filter((it) => it.id > seenUpTo.value));

const hide = () => invoke("hide_window");
const expand = async () => {
  await session.sendDraft("main", composer.value?.getText() ?? "");
  composer.value?.setText("");
  await invoke("show_main", { tab: "chat" });
};

// Cao đúng bằng nội dung (đo ở div bên trong, vì .bar luôn cao bằng cửa sổ nên không co lại được), tối đa 480 px.
let lastHeight = 0;
/** `force`: đặt lại cỡ dù chiều cao không đổi (mỗi lần cửa sổ hiện); còn lại bỏ qua lần trùng để khỏi gọi IPC thừa. */
async function fit(force = false) {
  await nextTick();
  const want = (inner.value?.offsetHeight ?? 60) + 2;
  capped.value = want > 480;
  const h = Math.min(480, Math.max(60, want));
  if (!force && h === lastHeight) return;
  lastHeight = h;
  await getCurrentWindow().setSize(new LogicalSize(680, h));
}

onMounted(async () => {
  await session.init();
  // Rust phát `bar-shown` sau khi đã chụp ngữ cảnh và hiện cửa sổ.
  await listen("bar-shown", async () => {
    seenUpTo.value = Math.max(0, ...session.items.value.map((i) => i.id));
    await session.reload().catch(() => undefined);
    await session.refreshContext();
    composer.value?.focus(true);
    await fit(true);
  });
  await listen("reload", () => session.reload());
  await session.onDraft((t) => composer.value?.setText(t));
  // Nội dung đổi cỡ vì bất kỳ lý do gì (ngữ cảnh tải xong, gợi ý hiện, font nạp...) thì co/giãn cửa sổ theo.
  if (inner.value) new ResizeObserver(() => void fit()).observe(inner.value);
  await fit(true);
});
watch([session.items, session.error, session.prompt, () => composer.value?.getText()], () => void fit(), { deep: true });
</script>

<template>
  <div class="bar" :class="{ scroll: capped }"><div ref="inner">
    <Composer ref="composer" :session="session" @hide="hide">
      <template #actions>
        <button class="btn small" title="Mở giao diện chính" @click="expand">Mở rộng</button>
      </template>
    </Composer>
    <div class="thread quick">
      <div v-for="it in visible" :key="it.id" class="entry">
        <div class="line" :title="'Bấm để điền lại'" @click="composer?.setText(it.line)">{{ it.line }}</div>
        <ResultView :item="it" :session="session" />
      </div>
    </div>
  </div></div>
</template>
