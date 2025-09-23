import { type RouteProps, useRouterData } from "../river.utils.tsx";

export function Dyn(props: RouteProps<"/__/:dyn">) {
	const routerData = useRouterData(props);

	return <div>{routerData().params.dyn}</div>;
}
