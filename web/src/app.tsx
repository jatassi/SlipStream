import { type QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, type RouterHistory, createBrowserHistory } from "@tanstack/react-router";
import { z } from "zod";

import { LiveSessionEnded } from "./api/gen/zod/slipStream.schemas.zod";
import { createQueryClient } from "./api/query-client";
import { routeTransitions } from "./app/route-transition";
import { type LiveStream, createLiveStream } from "./live/stream";
import type { SessionEndReason } from "./live/types";
import { createAppRouter } from "./router";
import { refreshSession, sessionQuery } from "./session/session";

const SessionEnded = LiveSessionEnded.extend({ type: z.literal("session-ended") });

export type AppInstance = ReturnType<typeof createApp>;

type Parts = {
  queryClient: QueryClient;
  router: ReturnType<typeof createAppRouter>;
  live: LiveStream;
};

// The pages that show a session. A tab whose session read already came back
// signed out still follows the others to Sign in from one of these, and stays
// put anywhere else.
const sessionPages = new Set<string>(["/_app", "/portal", "/new-pin"]);

function end({ queryClient, router, live }: Parts, reason: SessionEndReason): boolean {
  const session = queryClient.getQueryData(sessionQuery.queryKey);
  const onSessionPage = router
    .matchRoutes(router.state.location.pathname, router.state.location.search)
    .some(({ routeId }) => sessionPages.has(routeId));
  if (session === undefined || (session === null && !onSessionPage)) {
    return false;
  }
  const { href } = router.state.location;
  live.stop();
  queryClient.clear();
  void router.navigate({
    to: "/sign-in",
    search: reason === "expired" ? { reason, redirect: href } : { reason },
    replace: true,
  });
  return true;
}

// The stream has stopped itself. The gate reads the session again and sends
// the tab where the Account may now go, whose layout reconnects the stream.
// Resetting rather than clearing the cache refetches what stays on screen,
// whose observers clear() would leave on the removed queries. A session read
// already in flight began before the change, so it is cancelled, not joined.
async function regate({ queryClient, router }: Parts) {
  await queryClient.cancelQueries({ queryKey: sessionQuery.queryKey });
  await refreshSession(queryClient);
  await router.invalidate();
  await queryClient.resetQueries();
}

export function createApp(history: RouterHistory = createBrowserHistory()) {
  const channel = new BroadcastChannel("slipstream-session");
  const transitions = routeTransitions(history);
  const queryClient = createQueryClient(() => {
    endSession("expired");
  });
  const live = createLiveStream({
    queryClient,
    onSessionEnded: endSession,
    onAccessChanged: () => {
      void regate(parts);
    },
  });
  const router = createAppRouter({ queryClient, live, endSession }, history, transitions.types);
  const parts: Parts = { queryClient, router, live };

  function endSession(reason: SessionEndReason) {
    if (end(parts, reason)) {
      // oxlint-disable-next-line unicorn/require-post-message-target-origin -- a BroadcastChannel has no target origin
      channel.postMessage({ type: "session-ended", reason } satisfies z.input<typeof SessionEnded>);
    }
  }

  channel.addEventListener("message", (message: MessageEvent<unknown>) => {
    const parsed = SessionEnded.safeParse(message.data);
    if (parsed.success) {
      end(parts, parsed.data.reason);
    }
  });

  return {
    queryClient,
    router,
    live,
    endSession,
    dispose: () => {
      live.stop();
      channel.close();
      transitions.dispose();
    },
  };
}

export function App({ app }: { app: AppInstance }) {
  return (
    <QueryClientProvider client={app.queryClient}>
      <RouterProvider router={app.router} />
    </QueryClientProvider>
  );
}
