import {
	buildMutationURL,
	buildQueryURL,
	resolveBody,
	submit,
} from "river.now/client";
import {
	riverAppConfig,
	type MutationOutput,
	type MutationPattern,
	type MutationProps,
	type QueryOutput,
	type QueryPattern,
	type QueryProps,
} from "./river.gen.ts";

export const api = { query, mutate };

async function query<P extends QueryPattern>(props: QueryProps<P>) {
	return await submit<QueryOutput<P>>(
		buildQueryURL(riverAppConfig, props),
		{
			method: "GET",
			...props.requestInit,
		},
		props.options,
	);
}

async function mutate<P extends MutationPattern>(props: MutationProps<P>) {
	return await submit<MutationOutput<P>>(
		buildMutationURL(riverAppConfig, props),
		{
			method: "POST",
			...props.requestInit,
			body: resolveBody(props),
		},
		props.options,
	);
}
