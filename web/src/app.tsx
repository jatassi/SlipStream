import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, type RouterHistory, createBrowserHistory } from "@tanstack/react-router";
import { z } from "zod";

import { LiveSessionEnded } from "./api/gen/zod/slipStream.schemas.zod";
import { createQueryClient } from "./api/query-client";
import { routeTransitions } from "./app/route-transition";
import { createLiveStream } from "./live/stream";
import type { SessionEndReason } from "./live/types";
import { createAppRouter } from "./router";
import { sessionQuery } from "./session/session";

const SessionEnded = LiveSessionEnded.extend({ type: z.literal("session-ended") });

export type AppInstance = ReturnType<typeof createApp>;

export function createApp(history: RouterHistory = createBrowserHistory()) {
  const channel = new BroadcastChannel("slipstream-session");
  const transitions = routeTransitions(history);
  const queryClient = createQueryClient(() => {
    endSession("expired");
  });
  const live = createLiveStream({ queryClient, onSessionEnded: endSession });
  const router = createAppRouter({ queryClient, live, endSession }, history, transitions.types);

  function end(reason: SessionEndReason): boolean {
    if (!queryClient.getQueryData(sessionQuery.queryKey)) {
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

  function endSession(reason: SessionEndReason) {
    if (end(reason)) {
      // oxlint-disable-next-line unicorn/require-post-message-target-origin -- a BroadcastChannel has no target origin
      channel.postMessage({ type: "session-ended", reason } satisfies z.input<typeof SessionEnded>);
    }
  }

  channel.addEventListener("message", (message: MessageEvent<unknown>) => {
    const parsed = SessionEnded.safeParse(message.data);
    if (parsed.success) {
      end(parsed.data.reason);
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
