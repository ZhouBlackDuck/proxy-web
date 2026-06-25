import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const apiPort = process.env.API_PORT || '3000'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${apiPort}`,
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
