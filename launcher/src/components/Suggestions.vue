<script setup lang="ts">
import type { Command } from "../lib/manifest";

defineProps<{ items: Command[]; selected: number }>();
defineEmits<{ pick: [i: number] }>();
</script>

<template>
  <ul v-if="items.length" class="suggest">
    <li v-for="(c, i) in items" :key="c.name" :class="{ sel: i === selected }" @mousedown.prevent="$emit('pick', i)">
      <b>{{ c.name }}</b>
      <span class="title">{{ c.title }}</span>
      <span v-if="c.source === 'external'" class="tag" :title="c.file">ngoài</span>
      <span v-else-if="c.source === 'launcher'" class="tag">launcher</span>
      <span class="dim desc">{{ c.description }}</span>
    </li>
  </ul>
</template>
