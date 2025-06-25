import { Container, loadBalance } from "@cloudflare/containers";

export class SiteContainer extends Container {
	defaultPort = 8080;
	sleepAfter = "5m";
	envVars = { PORT: "8080" };
}

type Env = { SITE_CONTAINER: any };

export default {
	async fetch(request: Request, env: Env, ctx: any): Promise<Response> {
		const container = await loadBalance(env.SITE_CONTAINER, 3);
		return await container.fetch(request);
	},
};
