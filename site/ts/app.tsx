import { RiverRootOutlet } from "@sjc5/river/solid";
import "../css/tailwind-output.css";
import { type RouteProps, useLoaderData } from "./app_utils.ts";

export function App() {
	return (
		<main>
			<RiverRootOutlet />
		</main>
	);
}

export function Home(props: RouteProps<"/">) {
	return <p>{useLoaderData(props)}</p>;
}
