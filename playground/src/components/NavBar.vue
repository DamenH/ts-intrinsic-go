<script setup lang="ts">
import { useClipboard } from '@vueuse/core'
import * as monaco from 'monaco-editor'
import { dark } from '../composables/dark'
import { cmd, defaultFiles, files } from '../composables/state'

const { copy, copied } = useClipboard({ copiedDuring: 2000 })

function share() {
  copy(location.href)
}

function reset() {
  if (
    // eslint-disable-next-line no-alert
    globalThis.confirm(
      'Reset all files and command to their defaults?',
    )
  ) {
    monaco!.editor.getModels().forEach((model) => {
      if (model.uri.authority === 'model') return
      model.dispose()
    })
    files.value = defaultFiles()
    cmd.value = '--noEmit'
  }
}
</script>

<template>
  <div flex self-end gap1 border rounded-full>
    <button title="Share" nav-button @click="share">
      <div
        :class="copied ? 'i-ri:check-line text-green' : 'i-ri:share-line'"
        text-2xl
      />
    </button>

    <button title="Start Over" nav-button @click="reset">
      <div i-ri:refresh-line text-2xl />
    </button>

    <button nav-button @click="dark = !dark">
      <div dark:i-ri:moon-line i-ri:sun-line text-2xl />
    </button>

    <a
      nav-button
      href="https://github.com/DamenH/ts-intrinsic-go"
      target="_blank"
      rel="noopener"
      title="Source on GitHub"
    >
      <div i-ri:github-fill text-2xl />
    </a>
  </div>
</template>
