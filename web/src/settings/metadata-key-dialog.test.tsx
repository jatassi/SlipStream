import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MetadataProvider } from "../api/gen/model";
import { getListMetadataProvidersMockHandler } from "../api/gen/query/metadata/metadata.msw";
import { renderApp } from "../test/render-app";

const tmdb: MetadataProvider = {
  id: "tmdb",
  label: "TMDB",
  summary: "TMDB is working.",
  key: "bundled",
  state: "working",
  until: null,
  keySetAt: null,
  keyPage: "https://www.tmdb.invalid/settings/api",
  takesPin: false,
  pinSet: false,
  attribution: {
    text: "This product uses the TMDB API but is not endorsed or certified by TMDB",
    logo: "tmdb",
  },
  details: { lastSuccess: null, lastError: null, nodes: [] },
};

const tvdb: MetadataProvider = {
  ...tmdb,
  id: "tvdb",
  label: "TVDB",
  summary: "TVDB is working.",
  keyPage: "https://www.tvdb.invalid/dashboard/account/apikey",
  takesPin: true,
  attribution: { text: "Metadata from TVDB.", logo: null },
};

function keyServer(answer: (body: unknown) => Response) {
  const sent: unknown[] = [];
  return {
    sent,
    handlers: [
      getListMetadataProvidersMockHandler({ providers: [tmdb, tvdb] }),
      http.put("*/api/v1/metadata/providers/tvdb/key", async ({ request }) => {
        const body: unknown = await request.json();
        sent.push(body);
        return answer(body);
      }),
    ],
  };
}

const accepted = () => HttpResponse.json({ ...tvdb, key: "yours", pinSet: true });

async function openTvdb() {
  await userEvent.click(
    page.getByRole("article", { name: "TVDB" }).getByRole("button", { name: "Set key" }),
  );
  return page.getByRole("dialog", { name: "Set your TVDB key" });
}

test("TVDB's attribution shows as text beside TMDB's logo and text", async () => {
  renderApp({ path: "/settings/metadata", handlers: keyServer(accepted).handlers });
  const row = page.getByRole("article", { name: "TVDB" });
  await expect.element(row.getByText("Metadata from TVDB.")).toBeVisible();
  await expect.element(row.getByRole("img")).not.toBeInTheDocument();
  await expect
    .element(page.getByRole("article", { name: "TMDB" }).getByRole("img", { name: "TMDB" }))
    .toBeVisible();
});

test("a provider that takes a PIN offers one beside the key, and a provider that doesn't offers none", async () => {
  const server = keyServer(accepted);
  renderApp({ path: "/settings/metadata", handlers: server.handlers });
  await userEvent.click(
    page.getByRole("article", { name: "TMDB" }).getByRole("button", { name: "Set key" }),
  );
  const tmdbDialog = page.getByRole("dialog", { name: "Set your TMDB key" });
  await expect.element(tmdbDialog.getByRole("textbox", { name: "Key" })).toBeVisible();
  await expect.element(tmdbDialog.getByRole("textbox", { name: "PIN" })).not.toBeInTheDocument();
  await userEvent.click(tmdbDialog.getByRole("button", { name: "Cancel" }));

  const dialog = await openTvdb();
  await userEvent.fill(dialog.getByRole("textbox", { name: "Key" }), "my-key");
  await userEvent.fill(dialog.getByRole("textbox", { name: "PIN" }), "ABCD1234");
  await userEvent.click(dialog.getByRole("button", { name: "Test and save" }));
  await expect
    .element(page.getByText("SlipStream now calls TVDB with your key and PIN."))
    .toBeVisible();
  expect(server.sent).toEqual([{ key: "my-key", pin: "ABCD1234" }]);
});

test("an empty PIN sends the key alone", async () => {
  const server = keyServer(accepted);
  renderApp({ path: "/settings/metadata", handlers: server.handlers });
  const dialog = await openTvdb();
  await userEvent.fill(dialog.getByRole("textbox", { name: "Key" }), "my-key");
  await userEvent.click(dialog.getByRole("button", { name: "Test and save" }));
  await expect.element(page.getByText("SlipStream now calls TVDB with your key.")).toBeVisible();
  expect(server.sent).toEqual([{ key: "my-key" }]);
});
