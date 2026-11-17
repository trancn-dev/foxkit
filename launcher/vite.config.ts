import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "node:path";

const root = import.meta.dirname;

// Hai trang: index.html (ô lệnh nhanh, cửa sổ "bar") và main.html (giao diện chính, cửa sổ "main").
export default defineConfig({
  plugins: [vue()],
  clearScreen: false,
  server: { port: 1420, strictPort: true },
  build: {
    target: "es2022",
    rollupOptions: { input: { index: resolve(root, "index.html"), main: resolve(root, "main.html") } },
  },
  test: { include: ["src/**/*.test.ts"] },
});
