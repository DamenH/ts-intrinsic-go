import Vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  base: process.env.VITE_BASE ?? './',
  plugins: [Vue(), UnoCSS()],
  build: {
    chunkSizeWarningLimit: 2000,
    rollupOptions: {
      output: {
        manualChunks: {
          'monaco-editor': ['monaco-editor'],
          shiki: ['shiki'],
        },
      },
    },
  },
})
