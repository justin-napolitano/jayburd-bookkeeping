import { defineConfig } from "vitest/config";
export default defineConfig({
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.{ts,tsx}", "demo/**/*.test.{ts,tsx}"],
    globals: false,
  },
});
