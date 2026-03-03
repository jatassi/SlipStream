import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import path from "node:path"
import { defineConfig } from "vite"

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
    port: 3000,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            return
          }

          const vendorChunks: [string, string[]][] = [
            ["vendor-react", ["/react-dom/", "/react/", "use-sync-external-store", "@tanstack/react-store"]],
            ["vendor-router", ["@tanstack/react-router", "@tanstack/router-core", "@tanstack/history"]],
            ["vendor-query", ["@tanstack/react-query"]],
            ["vendor-ui", ["@base-ui", "lucide-react", "cmdk", "sonner", "vaul", "react-day-picker"]],
            ["vendor-forms", ["react-hook-form", "@hookform", "/zod/"]],
          ]

          for (const [chunk, patterns] of vendorChunks) {
            if (patterns.some((p) => id.includes(p))) {
              return chunk
            }
          }
        },
      },
    },
  },
})
