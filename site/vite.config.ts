import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import { riverVitePlugin } from "./frontend/river.gen.ts";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
	plugins: [solid(), riverVitePlugin(), tailwindcss()],
});
