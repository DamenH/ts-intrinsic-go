import { refDebounced } from '@vueuse/core'
import { computed, ref, watchEffect } from 'vue'
import { useSourceFile, type SourceFileMap } from './source-file'
import { atou, utoa } from './url'

const DEFAULT_TSCONFIG = {
  compilerOptions: {
    target: 'esnext',
    module: 'esnext',
    strict: true,
    esModuleInterop: true,
    outDir: 'dist',
    declaration: true,
    resolveJsonModule: true,
  },
}

const DEFAULT_MAIN = `// \`Intrinsic<Fun, Args>\` evaluates a plain JavaScript function at
// type-check time and turns its return value into a type literal.
// Pick an example from the buttons above, or edit below and watch the
// .types panel update.

const add = (a: number, b: number) => a + b;
const greet = (name: string) => \`hello, \${name}!\`;

type Sum = Intrinsic<typeof add, [2, 40]>;             // 42
type Greeting = Intrinsic<typeof greet, ["world"]>;    // "hello, world!"

// Larger programs work too. See the Zod-like validator and SQL parser
// examples for functions that return structured types.
`

export const defaultFiles = (): SourceFileMap =>
  new Map([
    ['main.ts', useSourceFile('main.ts', DEFAULT_MAIN)],
    [
      'tsconfig.json',
      useSourceFile(
        'tsconfig.json',
        JSON.stringify(DEFAULT_TSCONFIG, undefined, 2),
      ),
    ],
  ])

export const cmd = ref('--noEmit')
export const watchMode = ref(false)
export const files = ref<SourceFileMap>(defaultFiles())
export const tabs = computed(() => Array.from(files.value.keys()))
export const activeFile = ref<string>('main.ts')

export const outputFiles = ref<Record<string, string | null>>({})
export const outputActive = ref<string | undefined>()

export interface ResolvedEntry {
  kind: 'type' | 'const'
  name: string
  type: string
}

export const resolvedTypes = ref<Map<string, Map<number, ResolvedEntry[]>>>(
  new Map(),
)

export function parseDumpedTypes(raw: string) {
  // Format, produced by cmd/tsgo/dumptypes.go:
  //   === /app/zod_test.ts ===
  //     L14: type T01 = "hello"
  //     L19: type T03 = 42
  //     L5:  const a1: 30
  const result = new Map<string, Map<number, ResolvedEntry[]>>()
  let currentMap: Map<number, ResolvedEntry[]> | undefined
  const push = (line: number, entry: ResolvedEntry) => {
    if (!currentMap) return
    const existing = currentMap.get(line)
    if (existing) existing.push(entry)
    else currentMap.set(line, [entry])
  }
  for (const line of raw.split('\n')) {
    const header = line.match(/^===\s*(.+?)\s*===$/)
    if (header) {
      // Strip /app/ prefix so lookups key by bare filename.
      const currentFile = header[1].replace(/^\/app\//, '')
      currentMap = new Map()
      result.set(currentFile, currentMap)
      continue
    }
    if (!currentMap) continue
    const typeAlias = line.match(/^\s*L(\d+):\s*type\s+(\S+)\s*=\s*(.+?)\s*$/)
    if (typeAlias) {
      push(Number(typeAlias[1]), {
        kind: 'type',
        name: typeAlias[2],
        type: typeAlias[3],
      })
      continue
    }
    const constDecl = line.match(/^\s*L(\d+):\s*const\s+(\S+):\s*(.+?)\s*$/)
    if (constDecl) {
      push(Number(constDecl[1]), {
        kind: 'const',
        name: constDecl[2],
        type: constDecl[3],
      })
    }
  }
  return result
}

export const compiling = ref(false)
export const timeCost = ref(0)
export const loadingWasm = ref(false)
export const initted = ref(false)

export const loading = computed(() => loadingWasm.value || !initted.value)
export const loadingDebounced = refDebounced(loading, 100)

export function filesToObject() {
  return Array.from(files.value.values()).map((file) => [
    file.filename,
    file.code,
  ])
}

const LAST_STATE_KEY = 'tsgo-intrinsic:state'
const serializedUrl = atou(location.hash!.slice(1))
let state = serializedUrl && JSON.parse(serializedUrl)
if (!state) {
  const serialized = localStorage.getItem(LAST_STATE_KEY)
  if (serialized) state = JSON.parse(serialized)
}
if (state) {
  try {
    cmd.value = state.c ?? '--noEmit'
    files.value = new Map(
      ((state?.f || []) as [string, string][]).map(([filename, code]) => [
        filename,
        useSourceFile(filename, code),
      ]),
    )
    if (files.value.size === 0) {
      files.value = new Map(defaultFiles())
    }
    activeFile.value = files.value.keys().next().value!
    watchMode.value = state.w || false
  } catch {}
}

export const serialized = computed(() =>
  JSON.stringify({
    f: filesToObject(),
    c: cmd.value,
    w: watchMode.value,
  }),
)

watchEffect(() => {
  location.hash = utoa(serialized.value)
  localStorage.setItem(LAST_STATE_KEY, serialized.value)
})

export function loadExample(
  entry: string,
  exampleFiles: Record<string, string>,
) {
  // Dispose old Monaco models so URIs don't collide with the new set.
  for (const file of files.value.values()) {
    file.dispose()
  }
  // Map insertion order drives the tab strip order. Put the entry
  // file first, then the rest in their original order, then tsconfig.
  const names = Object.keys(exampleFiles)
  const ordered = [entry, ...names.filter((n) => n !== entry)]
  const newFiles = new Map<string, ReturnType<typeof useSourceFile>>()
  for (const name of ordered) {
    newFiles.set(name, useSourceFile(name, exampleFiles[name]))
  }
  newFiles.set(
    'tsconfig.json',
    useSourceFile('tsconfig.json', JSON.stringify(DEFAULT_TSCONFIG, undefined, 2)),
  )
  files.value = newFiles
  activeFile.value = exampleFiles[entry] !== undefined
    ? entry
    : ordered[0]
}
