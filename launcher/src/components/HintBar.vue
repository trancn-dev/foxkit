<script setup lang="ts">
import { computed } from "vue";
import type { Example } from "../lib/manifest";
import type { Ctx } from "../lib/session";

const props = defineProps<{ ctx: Ctx; example: Example | null; syntax: string | null }>();

/** Chỉ lấy tên thư mục cuối cho gọn, giống nhau ở mọi app; tooltip là đường dẫn đầy đủ. */
function leaf(s: string): string {
  return s.replace(/[\\/]+$/, "").split(/[\\/]/).pop() || s;
}
const chip = computed(() => {
  if (props.ctx.source === "none" || !props.ctx.app) return null;
  return { text: props.ctx.dir ? `${props.ctx.app} · ${leaf(props.ctx.dir)}` : props.ctx.app, title: props.ctx.dir ?? props.ctx.title };
});
</script>

<template>
  <div class="hintbar">
    <span v-if="chip" class="chip" :title="chip.title">{{ chip.text }}</span>
    <span v-if="example" class="example" :title="example.desc">Ví dụ: {{ example.cmd }} — {{ example.desc }}</span>
    <span v-else-if="syntax" class="example">{{ syntax }}</span>
  </div>
</template>
