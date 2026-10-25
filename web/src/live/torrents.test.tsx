import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { DownloadclientsLiveRow } from "../api/gen/model";
import { listOf, torrent } from "../test/torrents";
import { renderKit } from "../test/render-kit";
import { splice } from "./torrents-splice";
import { takeTorrents, useLiveRow } from "./torrents";

const renders = new Map<string, number>();

function Speed({ id }: { id: string }) {
  const live = useLiveRow(id);
  renders.set(id, (renders.get(id) ?? 0) + 1);
  return <p aria-label={id}>{live === undefined ? "no sample" : `${live.downBps} B/s`}</p>;
}

const sample = (id: string, downBps: number): DownloadclientsLiveRow => ({
  id,
  downBps,
  upBps: 0,
  done: 0,
  wanted: 1,
  peers: 0,
  seeds: 0,
});

const queryClient = new QueryClient();

const tick = (rows: DownloadclientsLiveRow[]) => {
  takeTorrents(queryClient, {
    at: "2026-10-20T12:00:00Z",
    rows,
    downloads: { done: 0, wanted: 0, allPaused: false },
  });
};

test("a tick re-renders only the rows whose Torrents moved", async () => {
  const [moving, still] = ["live-moving", "live-still"];
  renderKit(
    <QueryClientProvider client={queryClient}>
      <Speed id={moving} />
      <Speed id={still} />
    </QueryClientProvider>,
  );
  tick([sample(still, 10)]);
  await expect.element(page.getByLabelText(still)).toHaveTextContent("10 B/s");
  const settled = renders.get(still);

  tick([sample(moving, 4096)]);
  await expect.element(page.getByLabelText(moving)).toHaveTextContent("4096 B/s");
  tick([sample(moving, 8192)]);
  await expect.element(page.getByLabelText(moving)).toHaveTextContent("8192 B/s");
  expect(renders.get(still)).toBe(settled);
  await expect.element(page.getByLabelText(still)).toHaveTextContent("10 B/s");
});

test("a splice replaces the named rows, drops the gone, appends the new and takes the counts", () => {
  const [kept, changed, gone] = [
    torrent({ state: "seeding" }),
    torrent({ state: "downloading" }),
    torrent({ state: "queued" }),
  ];
  const paused = { ...changed, state: "paused" } as const;
  const added = torrent({ state: "queued" });
  const fresh = listOf([kept, paused, added]);
  const next = splice(listOf([kept, changed, gone]), { ...fresh, rows: [paused, added] }, [
    changed.id,
    gone.id,
    added.id,
  ]);
  expect(next.rows).toEqual([kept, paused, added]);
  expect(next.counts).toEqual(fresh.counts);
});
