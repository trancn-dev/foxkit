<script setup lang="ts">
import { computed } from "vue";
import type { Example } from "../lib/manifest";
import type { Ctx } from "../lib/session";

const props = defineProps<{ ctx: Ctx; example: Example | null; syntax: string | null }>();

/** Cắt giữa đường dẫn quá dài; tooltip là đường dẫn đầy đủ. */
function middle(s: string, max = 42): string {
  return s.length <= max ? s : `${s.slice(0, 14)}…${s.slice(s.length - (max - 15))}`;
}
const chip = computed(() => {
  if (props.ctx.source === "none" || !props.ctx.app) return null;
  return { text: props.ctx.dir ? `${props.ctx.app} · ${middle(props.ctx.dir)}` : props.ctx.app, title: props.ctx.dir ?? props.ctx.title };
});
</script>

<template>
  <div class="hintbar">
    <span v-if="chip" class="chip" :title="chip.title">{{ chip.text }}</span>
    <span v-if="example" class="example" :title="example.desc">Ví dụ: {{ example.cmd }} — {{ example.desc }}</span>
    <span v-else-if="syntax" class="example">{{ syntax }}</span>
  </div>
</template>
