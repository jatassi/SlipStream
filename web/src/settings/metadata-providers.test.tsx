import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MetadataProvider } from "../api/gen/model";
import { getListMetadataProvidersMockHandler } from "../api/gen/query/metadata/metadata.msw";
import { renderApp } from "../test/render-app";
import { refused } from "../test/roots";

const tmdb: MetadataProvider = {
  id: "tmdb",
  label: "TMDB",
  summary: "TMDB is working.",
  key: "bundled",
  state: "working",
  until: null,
  keySetAt: null,
  keyPage: "https://www.tmdb.invalid/settings/api",
  attribution: {
    text: "This product uses the TMDB API but is not endorsed or certified by TMDB",
    logo: "tmdb",
  },
  details: { lastSuccess: null, lastError: null, nodes: [] },
};

const omdb: MetadataProvider = {
  ...tmdb,
  id: "omdb",
  label: "OMDb",
  summary: "OMDb is working.",
  key: "yours",
  keySetAt: "2026-05-03T09:00:00Z",
  keyPage: "https://www.omdb.invalid/apikey.aspx",
  attribution: null,
};

const providers = () => getListMetadataProvidersMockHandler({ providers: [tmdb, omdb] });

test("each provider shows its status sentence, whose key it uses and its key page", async () => {
  renderApp({ path: "/settings/metadata", handlers: [providers()] });
  const first = page.getByRole("article", { name: "TMDB" });
  await expect.element(first.getByText("TMDB is working.")).toBeVisible();
  await expect.element(first.getByText("SlipStream's key")).toBeVisible();
  await expect
    .element(first.getByRole("link", { name: "Get a TMDB key" }))
    .toHaveAttribute("href", tmdb.keyPage);
  await expect.element(first.getByRole("button", { name: "Remove key" })).not.toBeInTheDocument();

  const second = page.getByRole("article", { name: "OMDb" });
  await expect.element(second.getByText(/^Your key, set (3 May|May 3)$/u)).toBeVisible();
  await expect.element(second.getByRole("button", { name: "Change key" })).toBeVisible();
  await expect.element(second.getByRole("button", { name: "Remove key" })).toBeVisible();
  await expect.element(second.getByRole("img")).not.toBeInTheDocument();
});

test("TMDB's row carries the required attribution with TMDB's bundled logo", async () => {
  renderApp({ path: "/settings/metadata", handlers: [providers()] });
  const row = page.getByRole("article", { name: "TMDB" });
  await expect
    .element(
      row.getByText("This product uses the TMDB API but is not endorsed or certified by TMDB"),
    )
    .toBeVisible();
  const logo = row.getByRole("img", { name: "TMDB" });
  await expect.element(logo).toBeVisible();
  await expect
    .poll(() => {
      const img = logo.element();
      return img instanceof HTMLImageElement ? img.naturalWidth : 0;
    })
    .toBeGreaterThan(0);
});

test("Set key tests the key and saves it only once the provider accepts it", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: "/settings/metadata",
    handlers: [
      providers(),
      http.put("*/api/v1/metadata/providers/tmdb/key", async ({ request }) => {
        const body: unknown = await request.json();
        sent.push(body);
        return JSON.stringify(body) === '{"key":"good-key"}'
          ? HttpResponse.json({ ...tmdb, key: "yours", keySetAt: "2026-05-03T09:00:00Z" })
          : refused(
              422,
              "metadata_key_refused",
              "The metadata provider refused that key, so SlipStream didn't save it",
              {
                detail: "TMDB refused the key: Invalid API key: You must be granted a valid key.",
              },
            );
      }),
    ],
  });
  await userEvent.click(
    page.getByRole("article", { name: "TMDB" }).getByRole("button", { name: "Set key" }),
  );
  const dialog = page.getByRole("dialog", { name: "Set your TMDB key" });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Key" }), "wrong-key");
  await userEvent.click(dialog.getByRole("button", { name: "Test and save" }));
  await expect
    .element(
      dialog.getByText("The metadata provider refused that key, so SlipStream didn't save it"),
    )
    .toBeVisible();

  await userEvent.fill(dialog.getByRole("textbox", { name: "Key" }), "good-key");
  await userEvent.click(dialog.getByRole("button", { name: "Test and save" }));
  await expect.element(page.getByText("TMDB now uses your key.")).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent).toEqual([{ key: "wrong-key" }, { key: "good-key" }]);
});

test("Test calls the provider and shows the sentence it returns", async () => {
  let tested = 0;
  renderApp({
    path: "/settings/metadata",
    handlers: [
      providers(),
      http.post("*/api/v1/metadata/providers/omdb/test", () => {
        tested += 1;
        return HttpResponse.json({ ...omdb, summary: "OMDb refused the key. Set another." });
      }),
    ],
  });
  await userEvent.click(
    page.getByRole("article", { name: "OMDb" }).getByRole("button", { name: "Test" }),
  );
  await expect.element(page.getByText("OMDb refused the key. Set another.")).toBeVisible();
  expect(tested).toBe(1);
});

test("Remove key asks first and then removes the Admin's key", async () => {
  let removed = 0;
  renderApp({
    path: "/settings/metadata",
    handlers: [
      providers(),
      http.delete("*/api/v1/metadata/providers/omdb/key", () => {
        removed += 1;
        return HttpResponse.json({ ...omdb, key: "bundled", keySetAt: null });
      }),
    ],
  });
  await userEvent.click(
    page.getByRole("article", { name: "OMDb" }).getByRole("button", { name: "Remove key" }),
  );
  const dialog = page.getByRole("dialog", { name: "Remove your OMDb key?" });
  await userEvent.click(dialog.getByRole("button", { name: "Remove key" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(removed).toBe(1);
});
