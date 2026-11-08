import { type ComponentType, createElement, use } from "react";

import { UnreachableError } from "../api/errors";
import { getSystem } from "../api/gen/query/system/system";

export class PageDownloadError extends Error {
  override readonly name = "PageDownloadError";

  constructor(cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause), { cause });
  }
}

type Chunk<P> =
  | { readonly kind: "idle" }
  | { readonly kind: "loading"; readonly promise: Promise<ComponentType<P>> }
  | { readonly kind: "loaded"; readonly component: ComponentType<P> };

async function nodeResponds(): Promise<boolean> {
  try {
    await getSystem({ cache: "no-store" });
  } catch (error) {
    return !(error instanceof UnreachableError);
  }
  return true;
}

// TanStack's own lazyRouteComponent reloads the page when an import fails,
// and its preload resolves on failure.
export function lazyRouteComponent<P extends object, K extends string>(
  importer: () => Promise<Record<K, ComponentType<P>>>,
  exportName: K,
) {
  let chunk: Chunk<P> = { kind: "idle" };

  function load(): Promise<ComponentType<P>> {
    if (chunk.kind === "loaded") {
      return Promise.resolve(chunk.component);
    }
    if (chunk.kind === "loading") {
      return chunk.promise;
    }
    const promise = importer().then(
      (module) => {
        const component = module[exportName];
        chunk = { kind: "loaded", component };
        return component;
      },
      async (error: unknown) => {
        chunk = { kind: "idle" };
        throw (await nodeResponds())
          ? new PageDownloadError(error)
          : new UnreachableError({ cause: error });
      },
    );
    chunk = { kind: "loading", promise };
    return promise;
  }

  function Lazy(props: P) {
    return createElement(chunk.kind === "loaded" ? chunk.component : use(load()), props);
  }
  Lazy.preload = async (): Promise<void> => {
    await load();
  };
  return Lazy;
}
