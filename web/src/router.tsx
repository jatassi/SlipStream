import type { QueryClient } from "@tanstack/react-query";
import { type RouterHistory, createRouter } from "@tanstack/react-router";

import type { LiveStream } from "./live/stream";
import type { SessionEndReason } from "./live/types";
import { routeTree } from "./routeTree.gen";
import { RouteError } from "./shell/errors";

export type RouterContext = {
  queryClient: QueryClient;
  live: LiveStream;
  endSession: (reason: SessionEndReason) => void;
};

export function createAppRouter(
  context: RouterContext,
  history: RouterHistory,
  transitionTypes: () => string[] | false,
) {
  return createRouter({
    routeTree,
    context,
    history,
    defaultViewTransition: { types: transitionTypes },
    defaultPreload: "intent",
    defaultPreloadStaleTime: 0,
    // Components read the query cache their loader fills, so a revisit that
    // rendered before its loader reran would seed editors with what was cached.
    defaultStaleReloadMode: "blocking",
    defaultErrorComponent: RouteError,
    scrollRestoration: true,
  });
}

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof createAppRouter>;
  }
}
