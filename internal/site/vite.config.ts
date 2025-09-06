import tailwindcss from "@tailwindcss/vite";
import river from "river.now/vite";
import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import { riverViteConfig } from "./frontend/river.gen.ts";

export default defineConfig({
	plugins: [solid(), river(riverViteConfig), tailwindcss()],
});
