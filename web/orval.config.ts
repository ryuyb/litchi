import { defineConfig } from "orval";

export default defineConfig({
  litchi: {
    // Auto-generated from backend API handlers (swaggo/swag)
    input: "../docs/api/swagger.json",
    output: {
      target: "./src/api/generated.ts",
      schemas: "./src/api/schemas",
      client: "react-query",
      mode: "tags-split",
      mock: false,
      clean: true,
      prettier: false,
    },
    hooks: {
      afterAllFilesWrite: "pnpm exec biome check --write"
    },
  },
});