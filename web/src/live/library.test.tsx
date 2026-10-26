import type { InvalidateQueryFilters } from "@tanstack/react-query";
import { QueryClientProvider } from "@tanstack/react-query";
import { Suspense } from "react";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { WorksLibraryRow } from "../api/gen/model";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import {
  getListContentTypesMockHandler,
  getListLibraryMockHandler,
} from "../api/gen/query/works/works.msw";
import { createQueryClient } from "../api/query-client";
import { useLibrary } from "../library/projection";
import { frames } from "../test/frames";
// oxlint-disable-next-line import/max-dependencies -- the effects run against a real QueryClient, MSW and React
import { cinema, libraryRow, libraryServer, orchard, orchardRow } from "../test/library";
import { renderKit } from "../test/render-kit";
import { worker } from "../test/worker";
import { effects } from "./effects";
import { refresh } from "./refresh";
import { useRemoved } from "./removed";

const harvest = libraryRow({
  id: "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b",
  key: "9000002",
  label: "Orchard Harvest",
});
const ember = libraryRow({
  id: "6e8a0c2d-4f6b-4d8f-8a2c-4e6a8c0e2f4a",
  key: "9000003",
  label: "Ember Lane",
});

function Works() {
  return (
    <ul aria-label="Works">
      {useLibrary().map((work) => (
        <li key={work.id}>{work.label}</li>
      ))}
    </ul>
  );
}

function OrchardPage() {
  return <p>{useRemoved(orchardRow.id) ? "The Quiet Orchard was deleted" : "Open"}</p>;
}

function mount(rows: readonly WorksLibraryRow[]) {
  const library = libraryServer(rows);
  worker.use(library.handler, getListContentTypesMockHandler({ contentTypes: [cinema] }));
  const queryClient = createQueryClient(() => {});
  const ctx = {
    queryClient,
    refresh: (filters?: InvalidateQueryFilters) => {
      void refresh(queryClient, filters);
    },
  };
  renderKit(
    <QueryClientProvider client={queryClient}>
      <Suspense fallback={<p>Loading the Library</p>}>
        <Works />
        <OrchardPage />
      </Suspense>
    </QueryClientProvider>,
  );
  return { library, queryClient, ctx };
}

const works = () => page.getByRole("list", { name: "Works" }).getByRole("listitem");

test("a works.changed burst in one frame reads every named row in one request", async () => {
  const { library, ctx } = mount([orchardRow, harvest]);
  await expect
    .poll(() =>
      works()
        .elements()
        .map((row) => row.textContent),
    )
    .toEqual(["The Quiet Orchard", "Orchard Harvest"]);

  library.become([{ ...orchardRow, label: "The Loud Orchard" }, harvest, ember]);
  effects["works.changed"]({ workIds: [orchardRow.id] }, ctx);
  effects["works.changed"]({ workIds: [ember.id, orchardRow.id] }, ctx);
  effects["files.changed"]({ workId: harvest.id }, ctx);

  await expect
    .poll(() =>
      works()
        .elements()
        .map((row) => row.textContent),
    )
    .toEqual(["The Loud Orchard", "Orchard Harvest", "Ember Lane"]);
  expect(library.asked).toEqual([[orchardRow.id, ember.id, harvest.id]]);
});

test("ids named while a read is in flight go in the next read", async () => {
  const { library, ctx } = mount([orchardRow, harvest]);
  await expect.element(works().first()).toHaveTextContent("The Quiet Orchard");

  effects["works.changed"]({ workIds: [orchardRow.id] }, ctx);
  await frames(1);
  effects["works.changed"]({ workIds: [harvest.id] }, ctx);
  await expect.poll(() => library.asked).toEqual([[orchardRow.id], [harvest.id]]);
});

test("a grabs.changed for progress alone reads no row", async () => {
  const { library, ctx } = mount([orchardRow]);
  await expect.element(works().first()).toHaveTextContent("The Quiet Orchard");

  const grab = { grabId: "9e1b3d5f-7a9c-4e1b-8d3f-5a7c9e1b3d5f", workId: orchardRow.id };
  effects["grabs.changed"]({ ...grab, progress: true }, ctx);
  await frames(3);
  expect(library.asked).toEqual([]);
  effects["grabs.changed"]({ ...grab, progress: false }, ctx);
  await expect.poll(() => library.asked).toEqual([[orchardRow.id]]);
});

test("works.removed drops the row, forgets the Work's reads and tells its open page", async () => {
  const { library, queryClient, ctx } = mount([orchardRow, harvest]);
  await expect.element(page.getByText("Open")).toBeVisible();
  queryClient.setQueryData(getGetWorkQueryKey(orchardRow.id), orchard);

  library.become([harvest]);
  effects["works.removed"]({ workIds: [orchardRow.id] }, ctx);

  await expect
    .poll(() =>
      works()
        .elements()
        .map((row) => row.textContent),
    )
    .toEqual(["Orchard Harvest"]);
  await expect.element(page.getByText("The Quiet Orchard was deleted")).toBeVisible();
  expect(queryClient.getQueryData(getGetWorkQueryKey(orchardRow.id))).toBeUndefined();
  expect(library.asked).toEqual([]);
});

test("a Work deleted while its row is being read doesn't come back", async () => {
  const { ctx } = mount([orchardRow, harvest]);
  await expect.element(works().first()).toHaveTextContent("The Quiet Orchard");
  let release!: () => void;
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  const reads: string[] = [];
  worker.use(
    getListLibraryMockHandler(async ({ request }) => {
      reads.push(new URL(request.url).search);
      await held;
      return { rows: [orchardRow] };
    }),
  );

  effects["works.changed"]({ workIds: [orchardRow.id] }, ctx);
  await expect.poll(() => reads).toHaveLength(1);
  effects["works.removed"]({ workIds: [orchardRow.id] }, ctx);
  release();
  await frames(3);
  await expect
    .poll(() =>
      works()
        .elements()
        .map((row) => row.textContent),
    )
    .toEqual(["Orchard Harvest"]);
});

test("a Work deleted before its read starts isn't read", async () => {
  const { library, ctx } = mount([orchardRow, harvest]);
  await expect.element(works().first()).toHaveTextContent("The Quiet Orchard");

  effects["works.changed"]({ workIds: [orchardRow.id, harvest.id] }, ctx);
  effects["works.removed"]({ workIds: [orchardRow.id] }, ctx);
  await expect.poll(() => library.asked).toEqual([[harvest.id]]);
  expect(
    works()
      .elements()
      .map((row) => row.textContent),
  ).toEqual(["Orchard Harvest"]);
});

test("a targets.changed naming more Works than a splice reads the whole Library once", async () => {
  const { library, ctx } = mount([orchardRow]);
  await expect.element(works().first()).toHaveTextContent("The Quiet Orchard");

  const many = Array.from({ length: 1001 }, (_, index) =>
    libraryRow({
      id: `00000000-0000-4000-8000-${String(index).padStart(12, "0")}`,
      key: String(index),
      label: `Film ${String(index)}`,
    }),
  );
  library.become(many);
  effects["targets.changed"]({ workIds: many.map(({ id }) => id) }, ctx);
  await expect.poll(() => works().elements().length).toBe(1001);
  expect(library.asked).toEqual([]);
});
