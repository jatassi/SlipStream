import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderWithQualities } from "../test/qualities";
import { Checkbox } from "../ui/checkbox";
import { type Release, ReleaseName, ReleaseRow } from "./row";

const title = "The.Quiet.Orchard.2021.REPACK.1080p.WEB-DL.DDP5.1.DV.x265-FLUX";

const everything: Release = {
  title,
  facts: {
    quality: "webdl-1080p",
    revision: "v2 REPACK",
    hdr: "Dolby Vision",
    audio: "DD+ 5.1",
    group: "FLUX",
  },
  size: 6_657_199_308,
  freeleech: true,
};

test("a Release's name reads its facts as Go's words, then its title", async () => {
  renderWithQualities(<ReleaseName release={everything} />);
  await expect.element(page.getByText(title)).toBeVisible();
  const line = page.getByRole("button", { name: "WEBDL-1080p" }).element().closest("span.flex");
  expect(line?.textContent).toBe(
    "WEBDL-1080p · v2 REPACK · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB · Freeleech",
  );
});

test("the revision and Freeleech open their definitions", async () => {
  renderWithQualities(<ReleaseName release={everything} />);
  await expect
    .element(page.getByRole("button", { name: "v2 REPACK" }))
    .toHaveAccessibleDescription(
      "It's a later release of the same video that fixes a problem in an earlier one.",
    );
  await expect
    .element(page.getByRole("button", { name: "Freeleech" }))
    .toHaveAccessibleDescription("Downloading it doesn't count against your ratio on its Tracker.");
});

test("a fact the title doesn't name is left out", async () => {
  renderWithQualities(
    <ReleaseName
      release={{ title, facts: { quality: "bluray-1080p", group: "FKW" }, size: 10_522_669_875 }}
    />,
  );
  await expect.element(page.getByRole("button", { name: "Bluray-1080p" })).toBeVisible();
  expect(page.getByText("FKW · 9.8 GB").elements()).toHaveLength(1);
  expect(page.getByRole("button").elements()).toHaveLength(1);
});

test("a Release that shows no fact reads its title alone", async () => {
  renderWithQualities(<ReleaseName release={{ title: "The Quiet Orchard", facts: {} }} />);
  await expect.element(page.getByText("The Quiet Orchard")).toBeVisible();
  expect(page.getByRole("button").elements()).toHaveLength(0);
});

test("a row given where it was heard shows the title and the heard line, then the screen's words", async () => {
  renderWithQualities(
    <ul>
      <ReleaseRow
        release={everything}
        heard={{ indexers: ["TorrentLeech", "IPTorrents"], seeders: 42 }}
      >
        <p>Grabbed for 1080p.</p>
      </ReleaseRow>
    </ul>,
  );
  await expect.element(page.getByText(title)).toBeVisible();
  await expect.element(page.getByText("TorrentLeech and IPTorrents · 42 seeders")).toBeVisible();
  await expect.element(page.getByText("Grabbed for 1080p.")).toBeVisible();
});

test("a row with no heard line keeps the title off it and shows its action", async () => {
  renderWithQualities(
    <ul>
      <ReleaseRow release={everything} action={<button type="button">Grab</button>} />
    </ul>,
  );
  await expect.element(page.getByRole("button", { name: "Grab" })).toBeVisible();
  expect(page.getByText(title).elements()).toHaveLength(0);
});

test("a row's lead control is named by the Release's facts", async () => {
  renderWithQualities(
    <ul>
      <ReleaseRow
        release={{ title, facts: { quality: "bluray-1080p", group: "FKW" }, size: 10_522_669_875 }}
        renderLead={(factsId) => (
          <Checkbox labelledBy={factsId} checked onCheckedChange={() => {}} />
        )}
      />
    </ul>,
  );
  await expect
    .element(page.getByRole("checkbox", { name: "Bluray-1080p · FKW · 9.8 GB" }))
    .toBeChecked();
});
