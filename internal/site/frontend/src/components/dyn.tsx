import { type RouteProps, useRouterData } from "../app_utils.tsx";

export function Dyn(props: RouteProps<"/__/:dyn">) {
	const routerData = useRouterData(props);

	return <div>{routerData().params.dyn}</div>;
}
