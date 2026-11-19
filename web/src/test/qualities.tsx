import { QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, Suspense } from "react";

import { createQueryClient } from "../api/query-client";
import { renderKit } from "./render-kit";
import { targetHandlers } from "./targets";
import { worker } from "./worker";

/** Renders UI that names qualities, over a fresh cache that GET /qualities answers. */
export function renderWithQualities(ui: ReactNode, options?: Parameters<typeof renderKit>[1]) {
  worker.use(...targetHandlers());
  return renderKit(
    <QueryClientProvider client={createQueryClient(() => {})}>
      <Suspense fallback={null}>{ui}</Suspense>
    </QueryClientProvider>,
    options,
  );
}
