import { WasmFs } from '@wasmer/wasmfs'
import { tokenizeArgs } from 'args-tokenizer'
import { createBirpc } from 'birpc'

// Pre-populate globalThis.process before wasm-exec.js's IIFE sees it
// so the IIFE's `if (!globalThis.process)` fallback never installs
// its enosys-throwing stubs. tsgo calls os.Getwd() at startup via
// cmd/tsgo/sys.go, which routes through process.cwd() in js/wasm. If
// that throws, the whole program aborts before printing anything.
const PLAYGROUND_CWD = '/app'
let _cwd = PLAYGROUND_CWD
;(globalThis as unknown as { process: Record<string, unknown> }).process = {
  getuid: () => -1,
  getgid: () => -1,
  geteuid: () => -1,
  getegid: () => -1,
  getgroups: () => {
    throw new Error('not implemented')
  },
  pid: -1,
  ppid: -1,
  umask: () => {
    throw new Error('not implemented')
  },
  cwd: () => _cwd,
  chdir: (dir: string) => {
    _cwd = dir
  },
}

// Side-effect import: Go's wasm_exec.js is an IIFE that writes `Go` onto
// globalThis. Rollup refuses to statically resolve a named import from it,
// so we pick it up from the global scope after the side-effect runs.
import './wasm-exec.js'
interface GoInstance {
  importObject: WebAssembly.Imports
  argv: string[]
  env: Record<string, string>
  exit: (code: number) => void
  run: (instance: WebAssembly.Instance) => Promise<void>
}
const Go = (globalThis as unknown as { Go: new () => GoInstance }).Go
import type { MainFunctions } from './core.js'

const workerFunctions = {
  compile,
  watch,
  setSourceCode,
}
export type WorkerFunctions = typeof workerFunctions
const rpc = createBirpc<MainFunctions, WorkerFunctions>(workerFunctions, {
  post: (data) => postMessage(data),
  on: (fn) => addEventListener('message', ({ data }) => fn(data)),
})

export interface CompileResult {
  output: Record<string, string | null>
  time: number
}

// init
const go = new Go()
let wasmFs: WasmFs

async function compile(
  wasmMod: WebAssembly.Module,
  cmd: string,
  files: Record<string, string>,
): Promise<CompileResult> {
  mountFs(files)
  const args = tokenizeArgs(cmd)

  const startTime = performance.now()
  const code = await run(wasmMod, args)
  const compileOutput = await getOutputFiles(code)
  const time = performance.now() - startTime

  // Second pass: run tsgo --dumpTypes against the same source files so
  // the user can see what Intrinsic<> actually evaluated to. This runs
  // against /app which still has the files mounted from the first pass.
  const tsFiles = Object.keys(files).filter(
    (f) => f.endsWith('.ts') || f.endsWith('.tsx'),
  )
  let typesText: string | undefined
  try {
    // Remount so stdout/stderr start clean.
    mountFs(files)
    await run(wasmMod, ['--dumpTypes', ...tsFiles])
    const typesStdout = (await wasmFs.getStdOut()) as string
    if (typesStdout && typesStdout.trim().length > 0) {
      typesText = typesStdout
    }
  } catch {}

  // Put .types first so the output panel opens to it by default.
  const output: Record<string, string | null> = {}
  if (typesText) output['.types'] = typesText
  Object.assign(output, compileOutput)

  return {
    output,
    time,
  }
}

function setSourceCode(files: Record<string, string>) {
  wasmFs.volume.fromJSON(files, '/app')
}

function watch(
  wasmMod: WebAssembly.Module,
  cmd: string,
  files: Record<string, string>,
): Promise<number> {
  wasmFs = mountFs(files)
  const args = ['-w', ...tokenizeArgs(cmd)]

  const id = setInterval(async () => {
    rpc.setOutputFiles(await getOutputFiles())
  }, 500)

  return run(wasmMod, args).finally(() => {
    clearInterval(id)
    wasmFs = undefined!
  })
}

function mountFs(files: Record<string, string>) {
  wasmFs = new WasmFs()
  wasmFs.volume.fromJSON(files, '/app')
  // @ts-expect-error missing types for wasmFs.fs
  globalThis.fs = wasmFs.fs
  return wasmFs
}

async function getOutputFiles(code?: number) {
  const stdout = ((await wasmFs.getStdOut()) as string).trim()
  let stderr = await wasmFs.fs.readFileSync('/dev/stderr', 'utf8').trim()
  if (code != null && code !== 0) {
    stderr = `Exit code: ${code}\n\n${stderr}`.trim()
  }

  const output: Record<string, string | null> = {}
  if (stdout) output['<stdout>'] = stdout
  if (stderr) output['<stderr>'] = stderr
  Object.assign(output, wasmFs.volume.toJSON('/app/dist', undefined, true))

  return output
}

async function run(wasmMod: WebAssembly.Module, args: string[]) {
  const { promise, resolve } = Promise.withResolvers<number>()
  _cwd = PLAYGROUND_CWD
  go.exit = (code: number) => resolve(code)
  go.argv = ['js', ...args]
  go.env = { ...go.env, PWD: PLAYGROUND_CWD, HOME: PLAYGROUND_CWD }
  const instance = await WebAssembly.instantiate(wasmMod, go.importObject)

  await go.run(instance)
  return await promise
}
