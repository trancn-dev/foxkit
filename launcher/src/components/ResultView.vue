<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { columnsOf, copyForResult, tableItems, type Result } from "../lib/interpret";
import type { Session } from "../lib/session";
import type { ThreadItem } from "../lib/thread";

const props = defineProps<{ item: ThreadItem; session: Session; wide?: boolean }>();
const r = computed(() => props.item.result as Result | undefined);
const showJson = ref(false);
const copied = ref(false);
const logEl = ref<HTMLElement | null>(null);

const okData = computed(() => (r.value?.kind === "ok" ? r.value : null));
const rows = computed(() => tableItems(okData.value?.data));
const cols = computed(() => columnsOf(rows.value));
const kvEntries = computed(() => {
  const d = okData.value?.data;
  return d && typeof d === "object" && !Array.isArray(d) ? Object.entries(d as Record<string, unknown>) : [];
});

const isArrayOfObjects = (v: unknown): v is Record<string, unknown>[] => Array.isArray(v) && v.length > 0 && v.every((x) => x && typeof x === "object");
const show = (v: unknown): string => (v === null || v === undefined ? "" : typeof v === "object" ? JSON.stringify(v) : String(v));
const pretty = computed(() => JSON.stringify(okData.value?.data, null, 2));
const copyable = computed(() => (r.value ? copyForResult(r.value) !== null : false));

async function copy() {
  if (!r.value || !(await props.session.copyResult(r.value))) return;
  copied.value = true;
  setTimeout(() => (copied.value = false), 1500);
}

const streamLines = computed(() => (r.value?.kind === "stream" ? r.value.lines : []));
watch(streamLines, () => nextTick(() => logEl.value && (logEl.value.scrollTop = logEl.value.scrollHeight)));
</script>

<template>
  <div class="result" :class="{ wide }">
    <div v-if="item.status === 'running' && !r" class="dim">Đang chạy…</div>

    <template v-else-if="r?.kind === 'info'">
      <div>{{ r.text }}</div>
      <div v-for="(l, i) in r.lines" :key="i" class="dim mono">{{ l }}</div>
    </template>

    <template v-else-if="r?.kind === 'error'">
      <div class="err">{{ r.message }}</div>
      <div v-if="r.hint" class="dim">{{ r.hint }}</div>
      <pre v-if="r.detail" class="pre">{{ r.detail }}</pre>
      <button v-if="r.canAdmin && r.args" class="btn" @click="session.rerun(item, r.args!, true)">Chạy lại với quyền admin</button>
    </template>

    <template v-else-if="r?.kind === 'stream'">
      <pre ref="logEl" class="pre log">{{ r.lines.join("\n") }}</pre>
      <div class="row">
        <span class="dim">{{ r.running ? "Đang chạy…" : `Đã dừng (mã ${r.exit ?? "?"})` }}</span>
        <button v-if="r.running" class="btn" @click="session.stop()">Dừng</button>
      </div>
    </template>

    <template v-else-if="okData">
      <pre v-if="showJson" class="pre">{{ okData.raw || pretty }}</pre>
      <template v-else>
        <pre v-if="typeof okData.data === 'string'" class="pre">{{ okData.data }}</pre>
        <template v-else-if="okData.output === 'table'">
          <div v-if="rows.length === 0" class="dim">Không có kết quả</div>
          <div v-else class="scroll">
            <table>
              <thead><tr><th v-for="c in cols" :key="c">{{ c }}</th></tr></thead>
              <tbody><tr v-for="(row, i) in rows" :key="i"><td v-for="c in cols" :key="c">{{ show(row[c]) }}</td></tr></tbody>
            </table>
          </div>
        </template>
        <table v-else-if="okData.output === 'kv' && kvEntries.length" class="kv">
          <tr v-for="[k, v] in kvEntries" :key="k">
            <th>{{ k }}</th>
            <td>
              <table v-if="isArrayOfObjects(v)" class="sub">
                <thead><tr><th v-for="c in Object.keys(v[0])" :key="c">{{ c }}</th></tr></thead>
                <tbody><tr v-for="(row, i) in v" :key="i"><td v-for="c in Object.keys(v[0])" :key="c">{{ show(row[c]) }}</td></tr></tbody>
              </table>
              <template v-else>{{ show(v) }}</template>
            </td>
          </tr>
        </table>
        <pre v-else class="pre">{{ pretty }}</pre>
      </template>
      <div class="row">
        <button v-if="okData.dryRun && okData.applyArgs" class="btn primary" title="Ctrl+Enter" @click="session.rerun(item, okData.applyArgs!, false)">Áp dụng</button>
        <button v-if="copyable" class="btn" @click="copy">{{ copied ? "Đã copy" : "Copy" }}</button>
        <button class="btn" @click="showJson = !showJson">JSON</button>
      </div>
    </template>
  </div>
</template>
