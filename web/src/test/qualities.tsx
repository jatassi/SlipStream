import { QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, Suspense } from "react";

import { createQueryClient } from "../api/query-client";
import { renderKit } from "./render-kit";

/** Renders UI that names qualities over a fresh cache, which the base handlers' GET /qualities fills. */
export function renderWithQualities(ui: ReactNode, options?: Parameters<typeof renderKit>[1]) {
  return renderKit(
    <QueryClientProvider client={createQueryClient(() => {})}>
      <Suspense fallback={null}>{ui}</Suspense>
    </QueryClientProvider>,
    options,
  );
}
