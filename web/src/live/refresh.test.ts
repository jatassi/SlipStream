import { QueryClient } from "@tanstack/react-query";
import { expect, test, vi } from "vitest";

import { refresh } from "./refresh";

test("refresh waits out the fetch in flight of a query nobody watches, then invalidates it once", async () => {
  const queryClient = new QueryClient();
  const invalidate = queryClient.invalidateQueries.bind(queryClient);
  let invalidations = 0;
  vi.spyOn(queryClient, "invalidateQueries").mockImplementation(async (...args) => {
    invalidations += 1;
    if (invalidations > 10) {
      throw new Error("refresh spun without waiting for the fetch");
    }
    return invalidate(...args);
  });
  let fetched = 0;
  const loading = queryClient.fetchQuery({
    queryKey: ["work"],
    queryFn: async () => {
      fetched += 1;
      await new Promise((resolve) => {
        setTimeout(resolve, 50);
      });
      return fetched;
    },
  });

  await refresh(queryClient, { queryKey: ["work"] });
  await loading;

  expect({ invalidations, fetched }).toEqual({ invalidations: 1, fetched: 1 });
  expect(queryClient.getQueryState(["work"])?.isInvalidated).toBe(true);
});
