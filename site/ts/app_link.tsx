import { RiverLink } from "@sjc5/river/solid";

export function Link(props: Parameters<typeof RiverLink>[0]) {
	return <RiverLink prefetch="intent" {...props} />;
}
