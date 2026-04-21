# Custom Intrinsics Playground

A browser-hosted playground for the `Intrinsic<Fun, Args>` feature added on the
`custom-intrinsics` branch of this fork. The full `tsgo` compiler runs as a
WebAssembly module in a web worker. No server, no install.

Open [the live playground](https://damenh.github.io/ts-intrinsic-go/) to try it.

## Try it locally

```bash
# from the repo root
npx hereby playground:build   # compiles tsgo -> wasm, stages into playground/public
cd playground
pnpm install
pnpm dev
```

`pnpm build` produces `playground/dist/`, a fully static site deployable to
GitHub Pages or any static host.

## Examples

Example buttons in the top bar load snippets from `src/examples/` that
demonstrate the `Intrinsic<>` type: arithmetic, string transforms, control
flow, higher-order list operations, and larger showcases like a Zod-like
validator and a SQL parser, all evaluated at type-check time.

## Credits

This playground is forked from [sxzz/typescript-go-playground][sxzz-pg] by
Kevin Deng, which pioneered the "tsgo-as-wasm in a web worker" approach and
provided the Vue 3 / Monaco / `@wasmer/wasmfs` plumbing used here. The original
MIT license is preserved in [LICENSE](./LICENSE).

Changes from the upstream playground:

- Pins to a locally-built WASM from this branch instead of fetching daily
  builds from npm (so the custom-intrinsic changes are actually present)
- Drops the version/date selector
- Adds an **Examples** gallery wired to `src/examples/`

[sxzz-pg]: https://github.com/sxzz/typescript-go-playground
