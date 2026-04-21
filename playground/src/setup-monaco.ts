import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import { activeFile, files, resolvedTypes } from './composables/state'

// Expose monaco on window so the playwright probe can drive the
// editor without reaching into Vite's module scope.
;(globalThis as unknown as { monaco: typeof monaco }).monaco = monaco

monaco.typescript.typescriptDefaults.setCompilerOptions({
  target: monaco.typescript.ScriptTarget.ESNext,
  module: monaco.typescript.ModuleKind.ESNext,
  allowNonTsExtensions: true,
  allowImportingTsExtensions: true,
  moduleResolution: monaco.typescript.ModuleResolutionKind.NodeJs,
  noEmit: true,
  esModuleInterop: true,
  jsx: monaco.typescript.JsxEmit.Preserve,
  resolveJsonModule: true,
})

// Source of truth for cross-file diagnostics and intrinsic evaluation
// is the tsgo WASM worker. Turn off Monaco's semantic and suggestion
// diagnostics so it stops surfacing a second, wrong view on top of
// tsgo's output (no more `Cannot find name 'Intrinsic'`, no more
// `'T01' declared but never used`). Syntax validation stays on
// because it catches raw parse errors cheaply. Hovers and completions
// stay on because Monaco's worker drives normal identifier hovers.
monaco.typescript.typescriptDefaults.setDiagnosticsOptions({
  noSemanticValidation: true,
  noSuggestionDiagnostics: true,
  noSyntaxValidation: false,
})

// Shim so Monaco's TS worker can resolve `Intrinsic<...>` at all. It
// does not know about the `intrinsic` keyword from this fork's
// lib.es5. For type aliases where tsgo has a concrete resolved value,
// the hover provider below overrides the result.
monaco.typescript.typescriptDefaults.addExtraLib(
  `declare type Intrinsic<Fun, Args extends any[] = []> = unknown;`,
  'ts:intrinsic-shim.d.ts',
)

// Turn off Monaco's built-in hover registration. The provider below
// is the single voice in the widget: it either shows tsgo's resolved
// alias type or proxies to Monaco's TS worker via
// getTypeScriptWorker(). Without this we would get duplicated content
// and the unwanted `type T01 = Intrinsic<...huge union...>` preamble
// on top of the resolved type.
monaco.typescript.typescriptDefaults.setModeConfiguration({
  completionItems: true,
  hovers: false,
  documentSymbols: true,
  definitions: true,
  references: true,
  documentHighlights: true,
  rename: true,
  diagnostics: true,
  documentRangeFormattingEdits: true,
  signatureHelp: true,
  onTypeFormattingEdits: true,
  codeActions: true,
  inlayHints: true,
})

monaco.json.jsonDefaults.setDiagnosticsOptions({
  schemaValidation: 'warning',
  enableSchemaRequest: true,
  schemas: [
    {
      fileMatch: ['tsconfig.json'],
      uri: 'https://www.schemastore.org/tsconfig.json',
    },
  ],
})

monaco.editor.registerEditorOpener({
  openCodeEditor(_, resource) {
    if (resource.scheme !== 'file' || resource.path[0] !== '/') {
      return false
    }
    const path = resource.path.slice(1)
    if (!files.value.has(path)) {
      return false
    }
    activeFile.value = path
    return true
  },
})

// Split a type annotation on top-level `|` unions, respecting depth
// inside `<>`, `()`, `[]`, `{}`, and string literals. Needed because
// Monaco hands back flat strings like
// `const q: "a" | "b" | Record<string, any>` and we want to count and
// trim the union members without splitting inside nested generics.
function splitTopLevelUnion(type: string): string[] {
  const parts: string[] = []
  let depth = 0
  let start = 0
  let inString: string | null = null
  for (let i = 0; i < type.length; i++) {
    const ch = type[i]
    if (inString) {
      if (ch === '\\') {
        i++
        continue
      }
      if (ch === inString) inString = null
      continue
    }
    if (ch === '"' || ch === "'" || ch === '`') {
      inString = ch
      continue
    }
    if (ch === '<' || ch === '(' || ch === '[' || ch === '{') depth++
    else if (ch === '>' || ch === ')' || ch === ']' || ch === '}') depth--
    else if (ch === '|' && depth === 0) {
      parts.push(type.slice(start, i).trim())
      start = i + 1
    }
  }
  parts.push(type.slice(start).trim())
  return parts
}

// Replace wide unions (more than a handful of members) in a Monaco
// quick-info string with a shortened placeholder. Operates on any
// top-level `: <type>` in the text, so both `const q: A | B | ...`
// and function signatures like `(x: any) => A | B | ...` get trimmed.
function collapseWideUnion(text: string): string {
  const UNION_MEMBER_LIMIT = 5
  // Find the top-level `: ` that starts the type annotation, then the
  // end of the type. For const bindings this is the whole rest; for
  // function signatures it's trickier, but truncating at the end of
  // the string works because the worker's displayText is just one
  // declaration per call.
  const colon = text.search(/:\s/)
  if (colon === -1) return text
  const head = text.slice(0, colon + 2)
  const tail = text.slice(colon + 2)
  const members = splitTopLevelUnion(tail)
  if (members.length <= UNION_MEMBER_LIMIT) return text
  const shown = members.slice(0, 2).join(' | ')
  return `${head}${shown} | ... (${members.length - 2} more)`
}

// Unified hover provider. For type aliases and variable bindings
// that tsgo has resolved to a concrete type, show tsgo's answer.
// Everything else (imports, expressions, generic aliases that have
// not been instantiated, method calls) delegates to Monaco's TS
// worker so normal identifier hover still works.
monaco.languages.registerHoverProvider(['typescript', 'javascript'], {
  async provideHover(model, position) {
    // 1. tsgo override. Match by word at cursor, not just line, so
    // hovering `a1` in `const a1 = addTest(10, 20);` picks the `a1`
    // binding even though line 4 also contains `addTest`, `10`, `20`.
    const filename = model.uri.path.replace(/^\//, '')
    const fileMap = resolvedTypes.value.get(filename)
    const lineEntries = fileMap?.get(position.lineNumber)
    const word = model.getWordAtPosition(position)
    if (lineEntries && word) {
      const entry = lineEntries.find((e) => e.name === word.word)
      if (entry) {
        const range = new monaco.Range(
          position.lineNumber,
          word.startColumn,
          position.lineNumber,
          word.endColumn,
        )
        // Cases where tsgo evaluated the alias but its printed form is
        // not directly useful as hover content:
        //  - bare `intrinsic` sentinel: a generic intrinsic alias with
        //    no concrete arguments bound
        //  - circular print like `Foo = Foo<X>`: tsgo's printer prefers
        //    the alias name over expanding mapped types, conditional
        //    types, or other complex shapes
        // In both cases Monaco's TS worker would render something
        // misleading (the Intrinsic shim → `unknown`), so show our own
        // "deferred" message and stop. Returning here also blocks the
        // worker fallback below.
        const isCircular =
          entry.kind === 'type' &&
          new RegExp('^' + entry.name + '\\b').test(entry.type)
        const isDeferredSentinel =
          entry.kind === 'type' && entry.type === 'intrinsic'
        if (isDeferredSentinel || isCircular) {
          return {
            range,
            contents: [
              { value: `**${entry.name}** *(deferred type alias)*` },
              {
                value:
                  'tsgo cannot represent this alias as a concrete type ' +
                  'at the declaration site. Hover an instantiation like ' +
                  '`' +
                  entry.name +
                  '<...>` with concrete arguments to see the ' +
                  'evaluated result.',
              },
            ],
          }
        }
        // Skip the override when the resolved type embeds an
        // `intrinsic` token elsewhere (tsgo could not fully reduce a
        // function return or similar component). Fall through to
        // Monaco's worker so the user still sees a useful signature.
        if (!/\bintrinsic\b/.test(entry.type)) {
          const header =
            entry.kind === 'type'
              ? `type ${entry.name} = ${entry.type}`
              : `const ${entry.name}: ${entry.type}`
          return {
            range,
            contents: [
              { value: `**${entry.name}** *(resolved by tsgo)*` },
              { value: '```ts\n' + header + '\n```' },
            ],
          }
        }
      }
    }

    // 2. Fall back to Monaco's TS worker. This is what the stock
    // Monaco hover contribution does internally, so proxying gives us
    // the same normal-identifier hover the vanilla TS Playground has.
    try {
      const getWorker = await monaco.typescript.getTypeScriptWorker()
      const worker = await getWorker(model.uri)
      const offset = model.getOffsetAt(position)
      const info = await worker.getQuickInfoAtPosition(
        model.uri.toString(),
        offset,
      )
      if (!info) return null
      const rawDisplay =
        (info.displayParts ?? [])
          .map((p: { text: string }) => p.text)
          .join('') || ''
      // Monaco's quick-info for value-level calls at non-statically-
      // known sites (e.g. `const q1 = parseSql(s)`) returns the full
      // return-type union of the function, which can be dozens of
      // string literals long. If the type portion is a wide union,
      // collapse it to a readable placeholder. Signatures of function
      // types with embedded union returns get the same treatment.
      const displayText = collapseWideUnion(rawDisplay)
      const docs =
        (info.documentation ?? [])
          .map((p: { text: string }) => p.text)
          .join('') || ''
      const tags =
        (info.tags ?? [])
          .map(
            (t: { name: string; text?: Array<{ text: string }> }) =>
              `*@${t.name}* ${(t.text ?? []).map((x) => x.text).join('')}`,
          )
          .join('\n\n') || ''
      const start = model.getPositionAt(info.textSpan.start)
      const end = model.getPositionAt(info.textSpan.start + info.textSpan.length)
      const contents: Array<{ value: string }> = [
        { value: '```typescript\n' + displayText + '\n```' },
      ]
      if (docs) contents.push({ value: docs })
      if (tags) contents.push({ value: tags })
      return {
        range: new monaco.Range(
          start.lineNumber,
          start.column,
          end.lineNumber,
          end.column,
        ),
        contents,
      }
    } catch {
      return null
    }
  },
})

globalThis.MonacoEnvironment = {
  getWorker(_: any, label: string) {
    if (label === 'json') {
      return new jsonWorker()
    }
    if (label === 'typescript' || label === 'javascript') {
      return new tsWorker()
    }
    return new editorWorker()
  },
}
