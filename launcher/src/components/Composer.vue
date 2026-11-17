<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { findCommand } from "../lib/resolve";
import type { Result } from "../lib/interpret";
import type { Session } from "../lib/session";
import { tokenize } from "../lib/tokenize";
import ArgHint from "./ArgHint.vue";
import HintBar from "./HintBar.vue";
import Suggestions from "./Suggestions.vue";

const props = defineProps<{ session: Session }>();
/** `hide`: Esc hoặc Shift+Enter đã copy xong (ô nhanh sẽ ẩn). */
const emit = defineEmits<{ hide: [] }>();

const text = ref("");
const input = ref<HTMLInputElement | null>(null);
const sel = ref(0);
const caretEnd = ref(true);
const showSugg = ref(true);
const seed = ref(Math.random());

const s = props.session;
const resolved = computed(() => findCommand(tokenize(text.value).tokens, s.commands.value));
const sugg = computed(() => (showSugg.value ? s.suggestions(text.value) : []));
const ghost = computed(() => (caretEnd.value && !s.prompt.value ? s.ghostFor(text.value) : null));
const example = computed(() => s.exampleFor(text.value, (n) => Math.floor(seed.value * n)));
const syntaxLine = computed(() => s.syntaxFor(text.value));

watch(text, () => {
  sel.value = 0;
  showSugg.value = true;
  s.error.value = null;
});
watch(sugg, (v) => {
  if (sel.value >= v.length) sel.value = 0;
});

function updateCaret() {
  const el = input.value;
  caretEnd.value = !!el && el.selectionStart === el.value.length && el.selectionEnd === el.value.length;
}

function fillSuggestion(i: number) {
  const c = sugg.value[i];
  if (!c) return;
  text.value = c.name + " ";
  nextTick(() => input.value?.setSelectionRange(text.value.length, text.value.length));
}

function acceptGhost(): boolean {
  if (ghost.value?.acceptable) {
    text.value += ghost.value.text;
    return true;
  }
  return false;
}

/** Tab ở ô trống hoặc chỉ có tên lệnh: điền ví dụ mờ. */
function acceptExample(): boolean {
  const t = tokenize(text.value);
  if (example.value && (t.tokens.length === 0 || (resolved.value && resolved.value.used === t.tokens.length && !t.trailingSpace))) {
    text.value = example.value.cmd;
    return true;
  }
  return false;
}

async function run(copyAfter: boolean) {
  const line = text.value;
  const r = await s.submit(line, { copyAfter });
  if (!r.ran) return;
  text.value = "";
  if (copyAfter) {
    const { copied } = await r.done;
    if (copied) emit("hide");
  }
}

function lastResult(pred: (r: Result) => boolean) {
  for (let i = s.items.value.length - 1; i >= 0; i--) {
    const r = s.items.value[i].result as Result | undefined;
    if (r && pred(r)) return { item: s.items.value[i], r };
  }
  return null;
}

async function onKey(e: KeyboardEvent) {
  if (s.prompt.value) {
    if (e.key === "Enter" || e.key === "Escape") {
      e.preventDefault();
      s.answer(e.key === "Enter");
    }
    return;
  }
  if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "c") {
    if (window.getSelection()?.toString()) return; // đang bôi chọn chữ: để Ctrl+Shift+C mặc định
    e.preventDefault();
    const last = lastResult((r) => r.kind === "ok");
    if (last) await s.copyResult(last.r);
    return;
  }
  if (e.ctrlKey && e.key === "Enter") {
    e.preventDefault();
    const last = lastResult((r) => r.kind === "ok" && !!r.dryRun);
    if (last && last.r.kind === "ok" && last.r.applyArgs) await s.rerun(last.item, last.r.applyArgs, false);
    return;
  }
  switch (e.key) {
    case "Enter":
      e.preventDefault();
      if (sugg.value.length && !resolved.value) fillSuggestion(sel.value);
      else await run(e.shiftKey);
      break;
    case "Tab":
      e.preventDefault();
      if (sugg.value.length && !resolved.value) fillSuggestion(sel.value);
      else if (!acceptGhost()) acceptExample();
      break;
    case "ArrowRight":
      if (caretEnd.value && acceptGhost()) e.preventDefault();
      break;
    case "ArrowDown":
    case "ArrowUp":
      if (sugg.value.length) {
        e.preventDefault();
        const n = sugg.value.length;
        sel.value = (sel.value + (e.key === "ArrowDown" ? 1 : n - 1)) % n;
      }
      break;
    case "Escape":
      e.preventDefault();
      if (sugg.value.length) showSugg.value = false;
      else emit("hide");
      break;
  }
}

function focus(selectAll = false) {
  seed.value = Math.random();
  input.value?.focus();
  if (selectAll) input.value?.select();
}
function setText(t: string) {
  text.value = t;
  nextTick(() => focus());
}
const getText = () => text.value;

onMounted(() => focus());
defineExpose({ focus, setText, getText });
</script>

<template>
  <div class="composer">
    <HintBar :ctx="s.ctx.value" :example="example" :syntax="syntaxLine" />
    <div class="inputrow">
      <div class="field">
        <input
          ref="input"
          v-model="text"
          spellcheck="false"
          autocomplete="off"
          placeholder="Gõ lệnh, ví dụ: port 8000"
          @keydown="onKey"
          @keyup="updateCaret"
          @click="updateCaret"
          @input="updateCaret"
        />
        <div v-if="ghost" class="ghostlayer" aria-hidden="true">
          <span class="typed">{{ text }}</span><span class="ghost" :class="{ remind: !ghost.acceptable }">{{ ghost.text }}</span>
        </div>
      </div>
      <slot name="actions" />
    </div>
    <div v-if="s.prompt.value" class="prompt">{{ s.prompt.value.text }}</div>
    <div v-else-if="s.error.value" class="error">
      {{ s.error.value.message }}<span v-if="s.error.value.hint" class="dim"> — {{ s.error.value.hint }}</span>
    </div>
    <div v-else-if="s.loadError.value" class="error">
      {{ s.loadError.value.message }}<span v-if="s.loadError.value.hint" class="dim"> — {{ s.loadError.value.hint }}</span>
    </div>
    <ArgHint v-if="!sugg.length && !s.error.value && !s.prompt.value" :input="text" :commands="s.commands.value" />
    <Suggestions :items="sugg" :selected="sel" @pick="fillSuggestion" />
  </div>
</template>
