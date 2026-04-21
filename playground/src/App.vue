<script setup lang="ts">
import { useClipboard, watchDebounced } from '@vueuse/core'
import AnsiRegex from 'ansi-regex'
import { watch } from 'vue'
import CodeEditor from './components/CodeEditor.vue'
import NavBar from './components/NavBar.vue'
import Tabs from './components/Tabs.vue'
import { dark } from './composables/dark'
import { examples } from './composables/examples'
import { shiki, themeDark, themeLight } from './composables/shiki'
import { useSourceFile } from './composables/source-file'
import {
  activeFile,
  cmd,
  compiling,
  files,
  filesToObject,
  initted,
  loadExample,
  loadingDebounced,
  loadingWasm,
  outputActive,
  outputFiles,
  parseDumpedTypes,
  resolvedTypes,
  serialized,
  tabs,
  timeCost,
  watchMode,
} from './composables/state'
import {
  createTsgoCli,
  getWorker,
  loadWasm,
  releaseWorker,
  terminateWorkers,
} from './core'

const ansiRegex = AnsiRegex()

let wasmMod: WebAssembly.Module | undefined

watchDebounced([files, cmd, watchMode], () => compile(), {
  debounce: 200,
  deep: true,
  immediate: true,
})

async function compile() {
  if (loadingWasm.value) return

  const current = serialized.value

  if (!wasmMod) {
    loadingWasm.value = true
    try {
      wasmMod = await loadWasm()
    } finally {
      loadingWasm.value = false
      initted.value = true
    }
  }

  if (current !== serialized.value) return
  if (watchMode.value && compiling.value) return

  compiling.value = true
  terminateWorkers()

  const worker = getWorker()
  const cli = createTsgoCli(worker, (f) => {
    outputFiles.value = f
  })

  if (watchMode.value) {
    const stop = watch(
      files,
      () => cli.setSourceCode(Object.fromEntries(filesToObject())),
      { deep: true },
    )

    await cli
      .watch(wasmMod, cmd.value, Object.fromEntries(filesToObject()))
      .finally(() => {
        releaseWorker(worker, cli)
        stop()
      })
    return
  }

  const result = await cli
    .compile(wasmMod, cmd.value, Object.fromEntries(filesToObject()))
    .finally(() => releaseWorker(worker, cli))
  if (current !== serialized.value) return

  compiling.value = false
  outputFiles.value = result.output
  timeCost.value = result.time
  outputActive.value = Object.keys(result.output)[0]

  const typesRaw = result.output['.types']
  resolvedTypes.value = typesRaw ? parseDumpedTypes(typesRaw) : new Map()
}

function highlight(code?: string | null) {
  if (!code) return ''
  return shiki.codeToHtml(code.replace(ansiRegex, ''), {
    lang: 'js',
    theme: dark.value ? themeDark.name! : themeLight.name!,
  })
}

const { copy, copied } = useClipboard()
function handleCopy() {
  if (!outputActive.value) return
  copy(outputFiles.value[outputActive.value] || '')
}

function addTab(name: string) {
  files.value.set(name, useSourceFile(name, ''))
}

function renameTab(oldName: string, newName: string) {
  files.value = new Map(
    Array.from(files.value.values()).map((file) => {
      if (file.filename === oldName) {
        file.rename(newName)
        return [newName, file]
      }
      return [file.filename, file]
    }),
  )
}

function removeTab(name: string) {
  files.value.get(name)?.dispose()
  files.value.delete(name)
}

function updateCode(name: string, code: string) {
  files.value.get(name)!.code = code
}
</script>

<template>
  <div flex="~ col" relative h-100dvh w-full px4 py2>
    <!-- Compact top bar: title, example buttons, nav icons -->
    <header flex items-center gap3 pb2>
      <div flex items-center gap2>
        <div i-catppuccin:typescript-test text-xl />
        <h1 text-base font-semibold>
          Custom Intrinsics
          <span op50 font-normal>Playground</span>
        </h1>
      </div>

      <div flex flex-wrap items-center gap1>
        <span op50 text-xs mr1>examples:</span>
        <button
          v-for="ex of examples"
          :key="ex.id"
          :title="ex.blurb"
          border
          rounded
          px2
          py="0.5"
          text-xs
          hover:bg-gray:10
          @click="loadExample(ex.entry, ex.files)"
        >
          {{ ex.label }}
        </button>
      </div>

      <div flex-1 />

      <NavBar />
    </header>

    <div v-if="loadingDebounced" flex-1 flex="~ col center" gap2 op70>
      <div i-logos:typescript-icon-round animate-pulse text-5xl />
      <div text-sm>
        Loading WASM compiler (~46 MB, cached after first load)…
      </div>
    </div>

    <!-- Editor + output split -->
    <div
      v-else
      min-h-0
      w-full
      flex="~ col"
      flex-1
      items-stretch
      gap2
      md:flex-row
    >
      <div flex="~ col" h-full min-w-0 w-full flex-1 gap1>
        <Tabs
          v-model="activeFile"
          :tabs
          h-full
          min-h-0
          min-w-0
          w-full
          flex-1
          @add-tab="addTab"
          @rename-tab="renameTab"
          @remove-tab="removeTab"
        >
          <template #default="{ value }">
            <div min-h-0 min-w-0 flex-1>
              <CodeEditor
                :model-value="files.get(value)!.code"
                :model="files.get(value)!.model"
                :uri="files.get(value)!.uri"
                input
                h-full
                min-h-0
                w-full
                @update:model-value="updateCode(value, $event)"
              />
            </div>
          </template>
        </Tabs>
        <div flex items-center gap2 text-xs font-mono>
          <span op60>❯ tsgo</span>
          <span v-if="watchMode" op60> -w</span>
          <input
            v-model="cmd"
            type="text"
            placeholder="flags, e.g. --noEmit"
            flex-1
            border
            rounded
            px2
            py="0.5"
          />
          <label flex items-center gap1 op70>
            <input v-model="watchMode" type="checkbox" /> watch
          </label>
        </div>
      </div>

      <div flex="~ col" h-full min-w-0 w-full flex-1 items-stretch gap1>
        <div
          v-if="compiling && !watchMode"
          flex="~ col"
          h-full
          w-full
          items-center
          justify-center
          gap2
        >
          <div i-logos:typescript-icon-round animate-bounce text-5xl />
          <span text-sm op70>Compiling…</span>
        </div>

        <Tabs
          v-else
          v-model="outputActive"
          :tabs="Object.keys(outputFiles)"
          readonly
          h-full
          min-h-0
          w-full
        >
          <div group relative h-full min-h-0 w-full>
            <template v-if="outputActive">
              <div
                v-if="outputActive.startsWith('<')"
                class="output"
                :class="{ 'text-red': outputActive === '<stderr>' }"
                v-text="outputFiles[outputActive]?.replace(ansiRegex, '')"
              />
              <div
                v-else
                class="output"
                v-html="highlight(outputFiles[outputActive])"
              />
            </template>
            <button
              absolute
              right-3
              top-3
              rounded
              p1
              op0
              transition-opacity
              hover:bg-gray
              hover:bg-opacity-10
              group-hover:opacity-100
              @click="handleCopy"
            >
              <div
                :class="
                  copied ? 'i-ri:check-line text-green' : 'i-ri:file-copy-line'
                "
              />
            </button>
          </div>
        </Tabs>

        <div v-if="timeCost && !compiling" self-end text-xs op50>
          {{ Math.round(timeCost) }} ms
        </div>
      </div>
    </div>
  </div>
</template>

<style>
.output {
  --at-apply: dark-bg-#1E1E1E w-full h-full overflow-scroll whitespace-pre
    border rounded p2 text-xs font-mono;
}
</style>
