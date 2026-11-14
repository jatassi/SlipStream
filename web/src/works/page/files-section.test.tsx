import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { problem } from "../../test/handlers";
import { orchardId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { scannedFiles } from "../../test/scanned-files";
import { workServer } from "../../test/work-server";

const files = () => page.getByRole("list", { name: "Files", exact: true });
const goneFiles = () => page.getByRole("list", { name: "Gone files" });
const forget = () => goneFiles().getByRole("button", { name: /^Forget/u });
const notifications = () => page.getByRole("region", { name: "Notifications" });
const goneId = scannedFiles.files[2]?.id;

test("a probed file shows its video, audio, subtitles and duration under Details", async () => {
  renderApp({
    path: `/works/${orchardId}`,
    handlers: workServer({ files: scannedFiles }).handlers,
  });
  const [uhd, hd] = [files().getByRole("listitem").nth(0), files().getByRole("listitem").nth(1)];
  await expect.element(uhd).toMatchTextContent(/2160p\.mkv/u);
  await expect.element(uhd.getByText("TrueHD Atmos · 7.1 · English")).not.toBeInTheDocument();

  await userEvent.click(uhd.getByRole("button", { name: "Details" }));

  for (const line of [
    "x265 · 3840×2160 · 10-bit · Dolby Vision profile 8, HDR10",
    "TrueHD Atmos · 7.1 · English",
    "DD · 5.1 · French",
    "English, French (forced)",
    "1 h 52 min",
  ]) {
    // oxlint-disable-next-line no-await-in-loop -- each line is checked on the same open Details
    await expect.element(uhd.getByText(line, { exact: true })).toBeVisible();
  }
  await expect.element(hd).toMatchTextContent(/1080p\.mkv/u);
  await expect.element(hd.getByRole("button", { name: "Details" })).not.toBeInTheDocument();
});

test("a gone file reads gone since the day the scan found it missing, and Forget forgets it", async () => {
  const server = workServer({ files: scannedFiles });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  await expect
    .element(goneFiles().getByRole("listitem"))
    .toMatchTextContent(/720p\.mkvmovies\/The Quiet Orchard \(2021\) · Gone since Nov 3/u);

  await userEvent.click(forget());

  await expect.element(goneFiles()).not.toBeInTheDocument();
  expect(server.forgot).toEqual([[goneId]]);
  expect(files().getByRole("listitem").all()).toHaveLength(2);
});

test("Forget on a file another tab already forgot shows no failure", async () => {
  let answered = false;
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      http.post("*/api/v1/files/forget", () => {
        answered = true;
        return problem(404, "file_not_found", "No file has that id");
      }),
      ...workServer({ files: scannedFiles }).handlers,
    ],
  });

  await userEvent.click(forget());

  await expect.poll(() => answered).toBe(true);
  await expect.element(forget()).not.toHaveAttribute("aria-busy");
  expect(notifications().element().textContent).toBe("");
});

test("a file back on disk can't be forgotten, and the toast says where it is", async () => {
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      http.post("*/api/v1/files/forget", () =>
        HttpResponse.json(
          {
            status: 409,
            code: "files_not_gone",
            title: "Only gone files can be forgotten",
            detail:
              "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 720p.mkv is on disk.",
          },
          { status: 409, headers: { "Content-Type": "application/problem+json" } },
        ),
      ),
      ...workServer({ files: scannedFiles }).handlers,
    ],
  });

  await userEvent.click(forget());

  await expect
    .element(notifications())
    .toHaveTextContent(
      "Failedmovies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 720p.mkv is on disk.",
    );
  await expect.element(goneFiles()).toBeVisible();
});
