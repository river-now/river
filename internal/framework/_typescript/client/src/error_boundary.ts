import type { RouteErrorComponent } from "./river_ctx/river_ctx.ts";

export const defaultErrorBoundary: RouteErrorComponent = (props: {
	error: string;
}) => {
	return "Route Error: " + props.error;
};
