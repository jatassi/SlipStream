import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addIndexerServer } from "../../test/add-indexer-server";
import { renderApp } from "../../test/render-app";

const feed = "http://jackett.lan:9117/api/v2.0/indexers/brook/results/torznab";
const posted = (writes: ReturnType<typeof addIndexerServer>["writes"]) =>
  writes.filter((write) => write.method === "POST").map((write) => write.body);

async function fillTorznab() {
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Brook");
  await userEvent.fill(page.getByRole("textbox", { name: "Feed address" }), feed);
  await userEvent.fill(page.getByLabelText("API key", { exact: true }), "jackett-key");
  await userEvent.click(page.getByRole("button", { name: "Test" }));
}

test("a Torznab feed is tested first; a feed that names no source asks for its Tracker", async () => {
  const server = addIndexerServer();
  renderApp({ path: "/settings/indexers/new/torznab", handlers: server.handlers });
  await expect
    .element(
      page.getByText("A Torznab feed reaches one Prowlarr or Jackett indexer.", { exact: false }),
    )
    .toBeVisible();
  await fillTorznab();
  await expect
    .element(page.getByText("The feed passed the Test and listed 50 items."))
    .toBeVisible();
  const tracker = page.getByRole("textbox", { name: "Tracker" });
  await expect.element(tracker).toHaveValue("Brook");
  await userEvent.fill(tracker, "Brook Tracker");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("heading", { name: "Brook", exact: true, level: 1 }))
    .toBeVisible();
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      kind: "torznab",
      address: feed,
      apiKey: "jackett-key",
      trackerName: "Brook Tracker",
    }),
  ]);
});

test("a feed with results from many sources warns that they share one Indexer's limits", async () => {
  const report = {
    items: 100,
    searchModes: [{ mode: "search" as const, params: ["q"] }],
    sources: ["Aither", "Brook", "Cinder"],
  };
  renderApp({
    path: "/settings/indexers/new/torznab",
    handlers: addIndexerServer({ report }).handlers,
  });
  await fillTorznab();
  await expect.element(page.getByText("Many sources")).toBeVisible();
  await expect
    .element(
      page.getByText(
        "This feed returned results from 3 sources. Each source becomes its own Tracker, and all of them share this Indexer's limits. Add one feed per site instead.",
      ),
    )
    .toBeVisible();
  expect(page.getByRole("textbox", { name: "Tracker" }).query()).toBeNull();
});

test("a failed Test shows Go's sentence, and changing the feed asks for a new Test", async () => {
  const failTest =
    "Brook refused the API key with 'Invalid API Key'. Check the account, then test again.";
  renderApp({
    path: "/settings/indexers/new/torznab",
    handlers: addIndexerServer({ failTest }).handlers,
  });
  await fillTorznab();
  await expect.element(page.getByRole("alert").getByText(failTest)).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Test" })).toBeVisible();
});

test("an RSS feed shows why the Test refused it, and saves once Allow items with no size is on", async () => {
  const refusal =
    "Some items have no size. Turn on Allow items with no size to use the feed anyway.";
  const server = addIndexerServer({ failSaves: [refusal] });
  renderApp({ path: "/settings/indexers/new/rss", handlers: server.handlers });
  await expect
    .element(
      page.getByText("An RSS feed lists a Tracker's newest items and is never searched.", {
        exact: false,
      }),
    )
    .toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Ferry RSS");
  await userEvent.fill(
    page.getByRole("textbox", { name: "Feed address" }),
    "https://ferry.example/rss?passkey=abc",
  );
  await userEvent.fill(page.getByRole("textbox", { name: "Tracker" }), "Ferry");
  await expect.element(page.getByRole("checkbox", { name: "Films" })).toBeChecked();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert").getByText(refusal)).toBeVisible();
  await userEvent.click(page.getByRole("checkbox", { name: "Allow items with no size" }));
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("heading", { name: "Ferry RSS", exact: true, level: 1 }))
    .toBeVisible();
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      kind: "rss",
      trackerName: "Ferry",
      media: ["cinema"],
      allowNoSize: false,
    }),
    expect.objectContaining({ allowNoSize: true }),
  ]);
});
