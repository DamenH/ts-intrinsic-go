import basics from '../examples/basics.ts?raw'
import openapi from '../examples/openapi.ts?raw'
import sql from '../examples/sql.ts?raw'
import zod from '../examples/zod.ts?raw'

export interface Example {
  id: string
  label: string
  blurb: string
  entry: string
  files: Record<string, string>
}

// Split a testdata fixture into virtual files on `// @filename:`
// directives. Harness option directives like `// @module:` are dropped
// because the playground supplies its own tsconfig.
function parseFixture(raw: string, fallback: string): Record<string, string> {
  const lines = raw.split('\n')
  const files: Record<string, string> = {}
  let current: string | undefined
  let buffer: string[] = []

  function flush() {
    if (current === undefined) return
    // trim trailing blank lines but keep internal structure
    while (buffer.length > 0 && buffer[buffer.length - 1] === '') buffer.pop()
    files[current] = buffer.join('\n') + '\n'
    buffer = []
  }

  for (const line of lines) {
    const fn = line.match(/^\/\/\s*@filename:\s*(.+)$/i)
    if (fn) {
      flush()
      current = fn[1].trim()
      continue
    }
    // Drop harness directives (module, target, moduleResolution, strict, etc.)
    if (/^\/\/\s*@[a-zA-Z]+\s*:/.test(line)) continue
    if (current === undefined) {
      // Content before any @filename lands in the fallback name.
      current = fallback
    }
    buffer.push(line)
  }
  flush()

  if (Object.keys(files).length === 0) {
    files[fallback] = raw
  }
  return files
}

function entryOf(files: Record<string, string>, preferred: string): string {
  // Testdata fixtures put the implementation first and the consumer
  // file (`*_test.ts`, `*_errors.ts`, `*_userland.ts`) last. The
  // consumer is what demonstrates the type-level results.
  const names = Object.keys(files)
  if (names.length > 1) return names[names.length - 1]
  if (files[preferred]) return preferred
  return names[0]
}

function makeExample(
  id: string,
  label: string,
  blurb: string,
  raw: string,
  preferred: string,
  entryOverride?: string,
): Example {
  const files = parseFixture(raw, preferred)
  // The override takes precedence over the "last file wins" heuristic.
  // Used by the basics example, where the tabs are peer topic files
  // with no canonical consumer.
  const entry =
    entryOverride && files[entryOverride]
      ? entryOverride
      : entryOf(files, preferred)
  return { id, label, blurb, entry, files }
}

export const examples: Example[] = [
  makeExample(
    'basics',
    'Basics',
    'Arithmetic, strings, comparisons, higher-order list ops, recursion, and control flow.',
    basics,
    'arithmetic.ts',
    'arithmetic.ts',
  ),
  makeExample(
    'zod',
    'Zod-like validator',
    'A runtime-shaped validator whose schema is inferred into a real TypeScript type at compile time.',
    zod,
    'zod.ts',
  ),
  makeExample(
    'sql',
    'SQL parser',
    'Parse a SQL string into a type-level AST. Demonstrates recursion, tokenization, and structured returns.',
    sql,
    'sql.ts',
  ),
  makeExample(
    'openapi',
    'OpenAPI schema',
    'Generate TypeScript types for OpenAPI operations entirely at type-check time.',
    openapi,
    'openapi.ts',
  ),
]
