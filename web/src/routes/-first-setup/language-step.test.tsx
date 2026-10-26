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

function open(place = metadataPlace()) {
  const server = firstSetup([done("address"), todo("language")]);
  renderApp({ path: "/setup/language", handlers: [...server.handlers, ...place.handlers] });
  return server;
}

const select = (name: string) => page.getByRole("combobox", { name });

test("language and region come from the browser while they hold the defaults, and Continue saves both at once", async () => {
  browserSpeaks("en-GB", "en");
  const metadata = metadataPlace();
  const server = open(metadata);
  await expect.element(stepHeading("Language and region")).toBeVisible();
  await expect
    .element(
      page.getByText(
        "Titles and overviews follow the language, and the region picks the certification and decides when a movie counts as out.",
      ),
    )
    .toBeVisible();
  await expect
    .element(page.getByText("File names keep the English title whatever language you pick."))
    .toBeVisible();
  await expect.element(select("Metadata language")).toHaveTextContent("English (United Kingdom)");
  await expect.element(select("Metadata region")).toHaveTextContent("United Kingdom");
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("First setup")).toBeVisible();
  expect(metadata.sent).toEqual([
    { values: { "metadata.language": "en-GB", "metadata.region": "GB" } },
  ]);
  expect(server.writes).toEqual([{ id: "language", outcome: "done" }]);
});

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
