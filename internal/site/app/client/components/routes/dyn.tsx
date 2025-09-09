import { type RouteProps, useRouterData } from "../app_utils.ts";

export function Dyn(props: RouteProps<"/__/:dyn">) {
	const routerData = useRouterData(props);

	return <div>{routerData().params.dyn}</div>;
}
