import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { downloading, failedImport } from "../../test/grabs";
import { orchardId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { failedUpgrade, upgrading } from "../../test/upgrade-cells";
import { workServer } from "../../test/work-server";

const cutRow = () =>
  page.getByRole("table", { name: "Editions and Targets" }).getByRole("row", { name: /^Any cut/u });
const missingRow = "Any cutFrom Profile UltraMissingSearchManual searchMissingSearchManual search";

test("an upgrade in flight reads downloading with the file it replaces and no Search", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  await expect.element(cutRow()).toHaveTextContent(missingRow);

  server.grabMoves(downloading, upgrading);

  await expect
    .element(cutRow())
    .toMatchTextContent(
      /^Any cutFrom Profile UltraMissingSearchManual searchDownloading25%.*Replaces HDTV-1080pManual search$/u,
    );
});

test("a failed upgrade reads failed with its cause and the file it kept", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  await expect.element(cutRow()).toHaveTextContent(missingRow);

  server.grabMoves(failedImport, failedUpgrade(failedImport.id));

  await expect
    .element(cutRow())
    .toHaveTextContent(
      "Any cutFrom Profile UltraMissingSearchManual search" +
        "FailedSlipStream couldn't put the file in the library.HDTV-1080p · 1920×1080 · 500 MBSearchManual search",
    );
  await expect
    .element(page.getByRole("region", { name: "Needs you" }).getByRole("article"))
    .toMatchTextContent(/^Failed at Import/u);
});
