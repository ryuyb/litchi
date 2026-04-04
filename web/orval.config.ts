import { defineConfig } from "orval";

export default defineConfig({
  litchi: {
    input: "./swagger.json",
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