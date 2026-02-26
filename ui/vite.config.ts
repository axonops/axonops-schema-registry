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
      "/subjects": "http://localhost:8081",
      "/schemas": "http://localhost:8081",
      "/config": "http://localhost:8081",
      "/mode": "http://localhost:8081",
      "/compatibility": "http://localhost:8081",
      "/contexts": "http://localhost:8081",
      "/admin": "http://localhost:8081",
      "/import": "http://localhost:8081",
      "/ui/auth": "http://localhost:8081",
      "/v1": "http://localhost:8081",
      "/metrics": "http://localhost:8081",
    },
  },
})
