import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  base: "/ui/",
  build: {
    outDir: "dist",
  },
  server: {
    proxy: {
      // All API calls go through the Go UI server
      "/api": "http://localhost:8080",
      "/health": "http://localhost:8080",
    },
  },
})
