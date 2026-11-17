<script setup lang="ts">
import { computed } from "vue";
import { label, type Command, type Param } from "../lib/manifest";
import { findCommand, syntax } from "../lib/resolve";
import { tokenize } from "../lib/tokenize";

const props = defineProps<{ input: string; commands: Command[] }>();

/** Tham số đang gõ: positional kế tiếp (bỏ qua cờ và giá trị cờ). */
const info = computed(() => {
  const { tokens, trailingSpace } = tokenize(props.input);
  const f = findCommand(tokens, props.commands);
  if (!f) return null;
  const rest = tokens.slice(f.used);
  const typing = !trailingSpace && rest.length > 0;
  const last = typing ? rest[rest.length - 1] : "";
  let param: Param | undefined;
  if (last.startsWith("--")) {
    param = f.cmd.flags.find((p) => p.name === last.slice(2).split("=")[0]);
  } else {
    let n = 0;
    for (let i = 0; i < (typing ? rest.length - 1 : rest.length); i++) {
      if (!rest[i].startsWith("--")) n++;
      else if (f.cmd.flags.find((p) => p.name === rest[i].slice(2))?.type !== "bool") i++;
    }
    param = typing ? f.cmd.args[Math.max(0, rest.filter((t) => !t.startsWith("--")).length - 1)] : f.cmd.args[n];
  }
  const detail = param
    ? [label(param), param.type, param.min !== undefined || param.max !== undefined ? `${param.min ?? ""}–${param.max ?? ""}` : "", param.options?.join("|") ?? ""].filter(Boolean).join(" · ")
    : "";
  return { syntax: syntax(f.cmd), detail, cmd: f.cmd };
});
</script>

<template>
  <div v-if="info" class="arghint">
    <code>{{ info.syntax }}</code>
    <span v-if="info.detail" class="dim"> — {{ info.detail }}</span>
  </div>
</template>
