<script setup lang="ts">
import { open } from "@tauri-apps/plugin-dialog";
import { computed, reactive, ref, watch } from "vue";
import { formToArgs, type FormValues } from "../lib/form";
import { label, type Command, type Param } from "../lib/manifest";
import { parseLine } from "../lib/resolve";
import type { Session } from "../lib/session";
import { quoteToken, tokenize } from "../lib/tokenize";
import ResultView from "../components/ResultView.vue";

const props = defineProps<{ cmd: Command; session: Session }>();

const values = reactive<FormValues>({ args: {}, flags: {} });

function reset() {
  values.args = {};
  values.flags = {};
  for (const f of props.cmd.flags) values.flags[f.name] = f.type === "bool" ? f.default === true : f.default !== undefined ? String(f.default) : "";
  for (const a of props.cmd.args) values.args[a.name] = a.multiple ? [""] : a.default !== undefined ? String(a.default) : "";
}
watch(() => props.cmd.name, reset, { immediate: true });

const built = computed(() => formToArgs(props.cmd, values));
const preview = computed(() => ["foxkit", ...built.value.args.map(quoteToken)].join(" "));
const shown = ref(false);

/** Lỗi hiện dưới đúng ô: thông báo bắt đầu bằng nhãn của tham số hoặc nhắc `--tên`. */
function fieldError(p: Param, isFlag: boolean): string | null {
  const e = built.value.error;
  if (!shown.value || !e) return null;
  return e.message.startsWith(label(p)) || (isFlag && e.message.includes(`--${p.name}`)) || e.message.includes(`Thiếu ${label(p)}`) ? e.message : null;
}
const generalError = computed(() => {
  const e = built.value.error;
  if (!shown.value || !e) return null;
  const all = [...props.cmd.args.map((p) => [p, false] as const), ...props.cmd.flags.map((p) => [p, true] as const)];
  return all.some(([p, f]) => fieldError(p, f)) ? null : e.message;
});

async function pick(p: Param, idx?: number) {
  const dir = p.type === "dir";
  const chosen = await open({ directory: dir, multiple: false });
  if (typeof chosen !== "string") return;
  if (p.multiple && idx !== undefined) (values.args[p.name] as string[])[idx] = chosen;
  else values.args[p.name] = chosen;
}

function fillExample() {
  const ex = props.cmd.examples?.[0];
  if (!ex) return;
  const parsed = parseLine(tokenize(ex.cmd).tokens, [props.cmd]);
  if (!parsed) return;
  reset();
  props.cmd.args.forEach((a, i) => {
    values.args[a.name] = a.multiple ? parsed.positional.slice(i) : (parsed.positional[i] ?? "");
  });
  for (const u of parsed.flags) values.flags[u.name] = u.value === true ? true : u.value;
}

async function runIt() {
  shown.value = true;
  if (built.value.error) return;
  await props.session.submitArgs(built.value.args);
}

const result = computed(() => props.session.items.value.find((i) => i.id === props.session.lastItemId.value));
</script>

<template>
  <div class="form">
    <h2>{{ cmd.title }} <span class="dim mono">{{ cmd.name }}</span></h2>
    <p class="dim">{{ cmd.description }}</p>

    <div v-for="a in cmd.args" :key="a.name" class="field-row">
      <label>{{ label(a) }}<b v-if="a.required"> *</b></label>
      <template v-if="a.multiple">
        <div v-for="(_, i) in values.args[a.name] as string[]" :key="i" class="inline">
          <input v-model="(values.args[a.name] as string[])[i]" />
          <button v-if="a.type === 'file' || a.type === 'dir'" class="btn small" @click="pick(a, i)">Chọn…</button>
        </div>
        <button class="btn small" @click="(values.args[a.name] as string[]).push('')">+ thêm dòng</button>
      </template>
      <template v-else>
        <textarea v-if="a.type === 'text'" v-model="values.args[a.name] as string" rows="3" />
        <select v-else-if="a.type === 'enum'" v-model="values.args[a.name] as string">
          <option value="">(không chọn)</option>
          <option v-for="o in a.options" :key="o" :value="o">{{ o }}</option>
        </select>
        <div v-else class="inline">
          <input v-model="values.args[a.name] as string" :type="a.type === 'int' ? 'number' : 'text'" :min="a.min" :max="a.max" />
          <button v-if="a.type === 'file' || a.type === 'dir'" class="btn small" @click="pick(a)">Chọn…</button>
        </div>
      </template>
      <div v-if="a.min !== undefined || a.max !== undefined" class="dim small">{{ a.min ?? "" }}–{{ a.max ?? "" }}</div>
      <div v-if="fieldError(a, false)" class="err small">{{ fieldError(a, false) }}</div>
    </div>

    <div v-for="f in cmd.flags" :key="f.name" class="field-row">
      <label v-if="f.type === 'bool'" class="check"><input v-model="values.flags[f.name] as boolean" type="checkbox" /> {{ label(f) }} <span class="dim mono">--{{ f.name }}</span></label>
      <template v-else>
        <label>{{ label(f) }} <span class="dim mono">--{{ f.name }}</span></label>
        <select v-if="f.type === 'enum'" v-model="values.flags[f.name] as string">
          <option value="">(mặc định)</option>
          <option v-for="o in f.options" :key="o" :value="o">{{ o }}</option>
        </select>
        <input v-else v-model="values.flags[f.name] as string" :type="f.type === 'int' ? 'number' : 'text'" :min="f.min" :max="f.max" />
        <div v-if="f.min !== undefined || f.max !== undefined" class="dim small">{{ f.min ?? "" }}–{{ f.max ?? "" }}</div>
      </template>
      <div v-if="fieldError(f, true)" class="err small">{{ fieldError(f, true) }}</div>
    </div>

    <div class="preview mono" :title="'Dòng lệnh tương đương'">{{ preview }} <button class="btn small" @click="session.copyResult({ kind: 'info', text: preview })">copy</button></div>
    <div v-if="generalError" class="err">{{ generalError }}</div>
    <div v-if="session.prompt.value" class="prompt">
      {{ session.prompt.value.text }}
      <button class="btn small" @click="session.answer(true)">Có</button> <button class="btn small" @click="session.answer(false)">Không</button>
    </div>
    <div v-else-if="session.error.value" class="err">{{ session.error.value.message }}</div>
    <div class="row">
      <button class="btn primary" @click="runIt">Chạy</button>
      <button v-if="cmd.examples?.length" class="btn" @click="fillExample">Ví dụ</button>
    </div>
    <div v-if="result" class="thread"><div class="entry"><div class="line">{{ result.line }}</div><ResultView :item="result" :session="session" wide /></div></div>
  </div>
</template>
