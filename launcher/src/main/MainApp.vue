<script setup lang="ts">
// Giao diện chính, cửa sổ `main` (launcher.md 6.10). Bấm X chỉ ẩn về tray (Rust chặn sự kiện đóng).
import { listen } from "@tauri-apps/api/event";
import { onMounted, ref } from "vue";
import { Session } from "../lib/session";
import ChatTab from "./ChatTab.vue";
import CommandsTab from "./CommandsTab.vue";
import FoldersTab from "./FoldersTab.vue";
import SettingsTab from "./SettingsTab.vue";

const TABS = [
  { id: "chat", name: "Chat" },
  { id: "commands", name: "Lệnh" },
  { id: "folders", name: "Thư mục lệnh" },
  { id: "settings", name: "Cài đặt" },
] as const;
type TabId = (typeof TABS)[number]["id"];

const session = new Session("main");
const KEY = "foxkit-main-tab";
const saved = (() => {
  try {
    return localStorage.getItem(KEY) as TabId | null;
  } catch {
    return null; // WebView chặn localStorage thì mở tab mặc định
  }
})();
const tab = ref<TabId>(saved && TABS.some((t) => t.id === saved) ? saved : "chat");
const chat = ref<InstanceType<typeof ChatTab> | null>(null);

function go(id: TabId) {
  tab.value = id;
  try {
    localStorage.setItem(KEY, id);
  } catch {
    // không nhớ được tab, không sao
  }
}

onMounted(async () => {
  await session.init();
  await session.refreshContext(); // chỉ để đọc clipboard cho @clip; cửa sổ chính không có ngữ cảnh app
  await listen<{ tab: TabId }>("show-tab", (e) => {
    go(e.payload.tab);
    session.refreshContext();
    chat.value?.focus();
  });
  await listen("reload", () => session.reload());
  // tray "Quét lại thư mục lệnh" = gõ :scan ở cửa sổ chính
  await listen<string>("run-line", (e) => session.submit(e.payload));
});
</script>

<template>
  <div class="main">
    <nav>
      <button v-for="t in TABS" :key="t.id" :class="{ on: tab === t.id }" @click="go(t.id)">{{ t.name }}</button>
    </nav>
    <main>
      <ChatTab v-show="tab === 'chat'" ref="chat" :session="session" />
      <CommandsTab v-if="tab === 'commands'" :session="session" />
      <FoldersTab v-if="tab === 'folders'" :session="session" />
      <SettingsTab v-if="tab === 'settings'" :session="session" />
    </main>
  </div>
</template>
