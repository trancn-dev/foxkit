import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "node:path";

// Hai trang: index.html (ô lệnh nhanh, cửa sổ "bar") và main.html (giao diện chính, cửa sổ "main").
export default defineConfig({
  plugins: [vue()],
  clearScreen: false,
  server: { port: 1420, strictPort: true },
  build: {
    target: "es2022",
    rollupOptions: { input: { index: resolve(__dirname, "index.html"), main: resolve(__dirname, "main.html") } },
  },
  test: { include: ["src/**/*.test.ts"] },
});
