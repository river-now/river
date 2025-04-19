import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import { riverVitePlugin } from "./frontend/river.gen.ts";

export default defineConfig({
	plugins: [solid(), riverVitePlugin()],
});
