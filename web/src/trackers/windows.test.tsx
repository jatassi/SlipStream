import { http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderApp } from "../test/render-app";
import { seedpoolId, seedpoolRows, trackersServer } from "../test/trackers";
import { aitherTrackerId } from "../test/indexers";
import { asPhone } from "../test/viewport";
import { noRuleSentence } from "./words";

const seedingTab = `/settings/trackers/${seedpoolId}/seeding`;
const open = () => page.getByRole("list", { name: "Open hit and run windows" });
const history = () => page.getByRole("list", { name: "History" });
const rowOf = (list: ReturnType<typeof open>, name: string) =>
  list.getByRole("listitem").filter({ hasText: name });

function recorded() {
  const asked: string[] = [];
  return {
    asked,
    handler: http.get("*/api/v1/trackers/:id/windows", ({ request }) => {
      const url = new URL(request.url);
      asked.push(`${url.pathname}${url.search}`);
    }),
  };
}

const region = (name: string) => page.getByRole("region", { name });

test("the Seeding tab lists the windows still owed with their lanes, and a link to each torrent", async () => {
  const windows = recorded();
  const { router } = renderApp({
    path: seedingTab,
    handlers: [windows.handler, ...trackersServer().handlers],
  });
  const holdFast = rowOf(open(), "Hold.Fast.2024.1080p.WEB-DL");
  await expect.element(holdFast).toMatchTextContent("3 days 4 h");
  await expect
    .element(holdFast.getByRole("meter", { name: "Seeding time" }))
    .toHaveAttribute("aria-valuetext", "6 days 22 h, rule 10 days 2 h, goal 14 days");
  const saltLine = rowOf(open(), "Salt.Line.2022.1080p.WEB-DL");
  await expect.element(saltLine).toMatchTextContent("5 days");
  await expect.element(saltLine.getByText("At risk")).toBeVisible();
  await expect.element(saltLine).toMatchTextContent(/stopped seeding .+seedpool/u);
  await expect.element(saltLine).toMatchTextContent(/missed .+grabbed again/u);
  expect(windows.asked).toEqual([
    `/api/v1/trackers/${seedpoolId}/windows?list=open`,
    `/api/v1/trackers/${seedpoolId}/windows?list=history`,
  ]);
  await userEvent.click(holdFast.getByRole("link"));
  await expect
    .poll(() => router.state.location.pathname)
    .toBe(`/settings/trackers/${seedpoolId}/torrents/${seedpoolRows[0]?.id ?? ""}`);
});

test("the history lists met, missed and released windows a page at a time", async () => {
  const windows = recorded();
  renderApp({ path: seedingTab, handlers: [windows.handler, ...trackersServer().handlers] });
  await expect.element(rowOf(history(), "Low.Tide.2023.1080p.WEB-DL")).toMatchTextContent(/Met /u);
  expect(history().getByText("Ash.Road.2020.1080p.WEB-DL").query()).toBeNull();
  const more = page
    .getByRole("region", { name: "History" })
    .getByRole("button", { name: "Show more" });
  await userEvent.click(more);
  await expect
    .element(rowOf(history(), "Ash.Road.2020.1080p.WEB-DL"))
    .toMatchTextContent(/Missed .+1 day/u);
  await expect
    .element(rowOf(history(), "Gone.Quiet.2019.1080p.WEB-DL"))
    .toMatchTextContent(/Closed /u);
  await expect.element(more).not.toBeInTheDocument();
  expect(windows.asked.slice(-1)).toEqual([
    `/api/v1/trackers/${seedpoolId}/windows?list=history&offset=2`,
  ]);
});

test("a window missed and grabbed again stays in the history as its miss while it owes again", async () => {
  renderApp({ path: seedingTab, handlers: trackersServer().handlers });
  const missed = rowOf(history(), "Salt.Line.2022.1080p.WEB-DL");
  await expect.element(missed).toMatchTextContent(/Missed .+6 days.+grabbed again/u);
  expect(missed.getByText("At risk").query()).toBeNull();
  expect(missed.getByText(/Still needs /u).query()).toBeNull();
  await expect.element(rowOf(open(), "Salt.Line.2022.1080p.WEB-DL")).toMatchTextContent("5 days");
});

test("a Tracker with no windows shows each list empty, naming the Tracker", async () => {
  renderApp({
    path: `/settings/trackers/${aitherTrackerId}/seeding`,
    handlers: trackersServer().handlers,
  });
  await expect.element(region("Open hit and run windows")).toMatchTextContent("Aither");
  await expect.element(region("History")).not.toBeEmptyDOMElement();
  expect(open().query()).toBeNull();
  expect(history().query()).toBeNull();
});

test("a private Tracker with no rule says so once, in its header's attention line", async () => {
  await asPhone();
  renderApp({ path: `/settings/trackers/${aitherTrackerId}`, handlers: trackersServer().handlers });
  const noRule = page.getByText(noRuleSentence("Aither", "private"));
  await expect.element(noRule).toBeVisible();
  expect(noRule.elements()).toHaveLength(1);
});
