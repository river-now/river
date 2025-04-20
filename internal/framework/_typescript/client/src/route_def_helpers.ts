type Module = Promise<Record<string, any>>;
type Key<T extends Module> = keyof Awaited<T>;
type ComponentObj<M extends Module> = { module: M; export?: Key<M> };
type Register = <M extends Module>(
	pattern: string,
	component: ComponentObj<M>,
) => void;
type Component = <M extends Module>(r: ComponentObj<M>) => ComponentObj<M>;
export type RiverRoutes = { Register: Register; Component: Component };
