import { expect, onTestFinished, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { done, firstSetup, metadataPlace, stepHeading, todo } from "../../test/first-setup";
import { renderApp } from "../../test/render-app";

function browserSpeaks(...languages: string[]) {
  vi.spyOn(navigator, "languages", "get").mockReturnValue(languages);
  onTestFinished(() => {
    vi.restoreAllMocks();
  });
}

function open(place = metadataPlace(), language = todo("language")) {
  const server = firstSetup([done("address"), language]);
  renderApp({ path: "/setup/language", handlers: [...server.handlers, ...place.handlers] });
  return server;
}

const select = (name: string) => page.getByRole("combobox", { name });

test("a language and region the Admin already chose stay as they are", async () => {
  browserSpeaks("en-GB");
  open(metadataPlace("fr-FR", "FR"));
  await expect.element(select("Metadata language")).toHaveTextContent("French (France)");
  await expect.element(select("Metadata region")).toHaveTextContent("France");
});

test("Skip keeps English and the United States", async () => {
  browserSpeaks("de-DE");
  const metadata = metadataPlace();
  const server = open(metadata);
  await expect.element(select("Metadata language")).toHaveTextContent("German (Germany)");
  await userEvent.click(page.getByRole("button", { name: "Skip" }));
  await expect.element(stepHeading("First setup")).toBeVisible();
  expect(metadata.sent).toEqual([]);
  expect(server.writes).toEqual([{ id: "language", outcome: "skipped" }]);
});

test("Continue on a done language step moves on without saving the same place again", async () => {
  browserSpeaks("en-GB");
  const metadata = metadataPlace("fr-FR", "FR");
  const server = open(metadata, done("language"));
  await expect.element(select("Metadata language")).toHaveTextContent("French (France)");
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("First setup")).toBeVisible();
  expect(metadata.sent).toEqual([]);
  expect(server.writes).toEqual([]);
});
