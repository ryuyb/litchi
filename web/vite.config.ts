import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";

const config = defineConfig({
	plugins: [
		devtools(),
		tsconfigPaths({ projects: ["./tsconfig.json"] }),
		tailwindcss(),
		tanstackStart({
			// SPA mode: Generate static HTML shell without Node.js server
			// The shell is output to _shell.html and can be embedded in Go backend
			spa: {
				enabled: true,
			},
		}),
		viteReact(),
	],
	server: {
		proxy: {
			// Proxy API requests to backend server
			"/api": {
				target: "http://localhost:8080",
				changeOrigin: true,
			},
			// Proxy WebSocket connections to backend server
			"/ws": {
				target: "ws://localhost:8080",
				ws: true,
			},
			// Proxy Swagger UI to backend server
			"/swagger": {
				target: "http://localhost:8080",
				changeOrigin: true,
			},
		},
	},
});

export default config;
