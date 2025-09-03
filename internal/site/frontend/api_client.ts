import {
	buildMutationURL,
	buildQueryURL,
	submit,
	type MutationProps,
	type QueryProps,
	type RiverMutationOutput,
	type RiverMutationPattern,
	type RiverQueryOutput,
	type RiverQueryPattern,
} from "river.now/client";
import { riverAppConfig, type RiverApp } from "./river.gen.ts";

export const api = { query, mutate };

async function query<P extends RiverQueryPattern<RiverApp>>(
	props: QueryProps<RiverApp, P>,
) {
	return await submit<RiverQueryOutput<RiverApp, P>>(
		buildQueryURL(riverAppConfig, props),
		{
			method: "GET",
			...props.requestInit,
		},
		props.options,
	);
}

async function mutate<P extends RiverMutationPattern<RiverApp>>(
	props: MutationProps<RiverApp, P>,
) {
	return await submit<RiverMutationOutput<RiverApp, P>>(
		buildMutationURL(riverAppConfig, props),
		{
			method: "POST",
			...props.requestInit,
			body: JSON.stringify(props.input),
		},
		props.options,
	);
}
