import { QueryClientProvider } from "@tanstack/react-query";
import type { RequestHandler } from "msw";
import { type ReactNode, Suspense, useEffect } from "react";

import { createQueryClient } from "../api/query-client";
import { Toasts } from "../ui/toasts";
import type { Subject } from "../works/actions/actions";
import { cinema } from "./library";
import { renderKit } from "./render-kit";
import { worker } from "./worker";

/** The nth cinema Work for an action, from 1, the first being the Quiet Orchard. */
export const film = (n: number): Subject => ({
  id: `3c1e9a52-7b4d-4f0e-8a6c-${String(n).padStart(12, "0")}`,
  label: n === 1 ? "The Quiet Orchard" : `Film ${String(n)}`,
  type: cinema,
});

export const picked = (count: number): Subject[] =>
  Array.from({ length: count }, (_, index) => film(index + 1));

/** Renders an action's dialog over a fresh cache, with the toasts it says its outcome in. */
export async function renderAction(ui: ReactNode, handlers: RequestHandler[] = []) {
  worker.use(...handlers);
  const queryClient = createQueryClient(() => {});
  await new Promise<void>((resolve) => {
    function Subscribed() {
      useEffect(resolve, []);
      return null;
    }
    renderKit(
      <QueryClientProvider client={queryClient}>
        <Toasts />
        <Subscribed />
        <Suspense fallback={null}>{ui}</Suspense>
      </QueryClientProvider>,
    );
  });
  return queryClient;
}
