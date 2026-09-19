import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import path from "node:path"
import { defineConfig } from "vite"

// portless (and the Playwright harness) hand the dev server its port through PORT
// and point it at a backend that may sit behind the portless HTTPS proxy.
const port = Number(process.env.PORT ?? 3000)
const apiOrigin = process.env.SLIPSTREAM_API_ORIGIN ?? "http://localhost:8080"
const wsOrigin = apiOrigin.replace(/^http/, "ws")

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  optimizeDeps: {
    include: ["lucide-react"],
  },
  server: {
    host: true,
    port,
    proxy: {
      "/api": {
        target: apiOrigin,
        changeOrigin: true,
        secure: false,
      },
      "/ws": {
        target: wsOrigin,
        ws: true,
        changeOrigin: true,
        secure: false,
      },
    },
  },
  build: {
    chunkSizeWarningLimit: 600,
  },
})
