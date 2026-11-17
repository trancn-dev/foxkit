<script setup lang="ts">
import { computed, ref } from "vue";
import { search } from "../lib/fuzzy";
import type { Command } from "../lib/manifest";
import type { Session } from "../lib/session";
import CommandForm from "./CommandForm.vue";

const props = defineProps<{ session: Session }>();
const q = ref("");
const selected = ref<string>("");

const commands = computed(() => props.session.commands.value.filter((c) => c.source !== "launcher"));
const filtered = computed(() => (q.value.trim() ? search(q.value, commands.value, 100) : commands.value));
const groups = computed(() => {
  const m = new Map<string, Command[]>();
  for (const c of filtered.value) m.set(c.category, [...(m.get(c.category) ?? []), c]);
  return [...m.entries()];
});
const current = computed(() => commands.value.find((c) => c.name === selected.value));
</script>

<template>
  <div class="cmds">
    <aside>
      <input v-model="q" class="search" placeholder="Tìm lệnh…" spellcheck="false" />
      <div v-for="[cat, list] in groups" :key="cat">
        <div class="cat dim">{{ cat }}</div>
        <div v-for="c in list" :key="c.name" class="item" :class="{ on: c.name === selected }" @click="selected = c.name">
          <b>{{ c.name }}</b>
          <span v-if="c.source === 'external'" class="tag" :title="c.file">ngoài</span>
          <div class="dim small">{{ c.title }}</div>
        </div>
      </div>
      <div v-if="!groups.length" class="dim pad">Không có lệnh nào khớp.</div>
    </aside>
    <section>
      <CommandForm v-if="current" :key="current.name" :cmd="current" :session="session" />
      <div v-else class="dim pad">Chọn một lệnh ở bên trái.</div>
    </section>
  </div>
</template>
