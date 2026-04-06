import { defineConfig } from "orval";

export default defineConfig({
  litchi: {
    // Design document (web/swagger.json) - contains full schema definitions
    // Generated document (docs/api/swagger.json) - contains implemented endpoints only
    // Switch to docs/api/swagger.json after T6.1.1~T6.1.7 are complete
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