import { createBirpc, type BirpcReturn } from 'birpc'
import TsgoCliWorker from './tsgo-cli.worker?worker'
import type { WorkerFunctions } from './tsgo-cli.worker'

// Adapted from sxzz/typescript-go-playground (MIT). Upstream fetches a dated
// tarball from npm (tsgo-wasm). We ship a single WASM built from this branch
// at build time via `hereby playground:build`.

let wasmModPromise: Promise<WebAssembly.Module> | undefined

export function loadWasm(): Promise<WebAssembly.Module> {
  if (!wasmModPromise) {
    // BASE_URL can be "./" (local preview), "/" (root deploy), or
    // "/<repo>/" (gh-pages project page). URL concat handles all three
    // as long as we resolve against document.baseURI.
    const base = new URL(
      `${import.meta.env.BASE_URL}tsgo.wasm`.replace(/^\.\//, ''),
      document.baseURI,
    )
    // In dev, append a timestamp query so `hereby playground:wasm`
    // rebuilds are picked up without hand-clearing browser cache.
    // In prod, let the browser respect HTTP caching on the static
    // asset (one big download on first visit, cached after).
    if (import.meta.env.DEV) {
      base.searchParams.set('t', String(Date.now()))
    }
    const url = base.href
    wasmModPromise = fetch(url, {
      cache: import.meta.env.DEV ? 'no-store' : 'default',
    })
      .then((r) => {
        if (!r.ok) throw new Error(`Failed to fetch ${url}: ${r.status}`)
        return r.arrayBuffer()
      })
      .then((buf) => WebAssembly.compile(buf))
  }
  return wasmModPromise
}

export const availableWorkers = Array.from(
  { length: 4 },
  () => new TsgoCliWorker(),
)
export const pendingWorkers = new Set<Worker>()

export function terminateWorkers() {
  const length = pendingWorkers.size
  if (length === 0) return

  for (const worker of pendingWorkers) {
    worker.terminate()
  }
  pendingWorkers.clear()

  const newWorkers = Array.from({ length }, () => new TsgoCliWorker())
  availableWorkers.push(...newWorkers)
}

export function getWorker(): Worker {
  let worker = availableWorkers.shift()
  if (!worker) {
    worker = new TsgoCliWorker()
  }
  pendingWorkers.add(worker)
  return worker
}

export function releaseWorker(
  worker: Worker,
  cli?: BirpcReturn<WorkerFunctions, MainFunctions>,
) {
  cli?.$close()
  pendingWorkers.delete(worker)
  availableWorkers.unshift(worker)
}

export interface MainFunctions {
  setOutputFiles: (files: Record<string, string | null>) => void
}

export function createTsgoCli(
  worker: Worker,
  setOutputFiles: (files: Record<string, string | null>) => void,
): BirpcReturn<WorkerFunctions, MainFunctions> {
  return createBirpc<WorkerFunctions, MainFunctions>(
    { setOutputFiles },
    {
      post: (data) => worker.postMessage(data),
      on: (fn) => worker.addEventListener('message', ({ data }) => fn(data)),
      // birpc defaults to a 60 s timeout. The first SQL or OpenAPI
      // compile after a cold wasm load can take longer because the Go
      // runtime needs to instantiate twice (once for --noEmit, once
      // for --dumpTypes) and intrinsic evaluation across the larger
      // examples is heavy. Bump to five minutes so we never bail on a
      // legitimately slow compile.
      timeout: 5 * 60 * 1000,
    },
  )
}
