import { RiverLink } from "river.now/solid";

export function Link(props: Parameters<typeof RiverLink>[0]) {
	return <RiverLink prefetch="intent" {...props} />;
}
