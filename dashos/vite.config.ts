import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { nodePolyfills } from "vite-plugin-node-polyfills";
import path from "path";

export default defineConfig({
  base: "/",
  plugins: [
    nodePolyfills({
      include: ["buffer", "util"],
      globals: { Buffer: true, global: true, process: true },
    }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "dist",
  },
  server: {
    port: 5174,
    proxy: {
      "/api": {
        target: "http://localhost:8080", // dev only: always proxy to moneyclaw
        changeOrigin: true,
      },
    },
  },
});
