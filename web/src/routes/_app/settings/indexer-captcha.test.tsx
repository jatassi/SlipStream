import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { captchas } from "../../../test/catalog";
import {
  larkspur,
  larkspurId,
  prowlarr,
  torrentLeech,
  torrentLeechId,
} from "../../../test/indexers";
import { indexersServer } from "../../../test/indexers-server";
import { homeId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";

const pageOf = (id: string) => `/settings/indexers/${id}`;
const dialog = () => page.getByRole("dialog");
const image = (name: string) => page.getByRole("img", { name: `The captcha ${name} shows` });
const code = () => page.getByRole("textbox", { name: "Code" });
const moveHome = async () => {
  await userEvent.click(page.getByRole("button", { name: "Move" }));
  await userEvent.click(dialog().getByRole("combobox", { name: "Search from" }));
  await userEvent.click(page.getByRole("option", { name: "home" }));
};
const pastedTorrentLeech = {
  ...torrentLeech,
  session: { nodeId: torrentLeech.homeNodeId, pasted: true, earnedAt: "2026-10-26T08:00:00Z" },
};
const needsCaptcha = "Larkspur needs a captcha to sign in. Type the code and test again.";

test("a Test that meets a captcha shows its image, and the next Test answers its code", async () => {
  const server = indexersServer({ captchaTests: [needsCaptcha] });
  renderApp({ path: pageOf(larkspurId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect.element(page.getByRole("alert").getByText(needsCaptcha)).toBeVisible();
  await expect.element(image("Larkspur")).toHaveAttribute("src", captchas[0]?.image);
  await expect
    .element(
      page.getByText("Signing in to Larkspur needs the code in this image, which home fetched.", {
        exact: false,
      }),
    )
    .toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect.element(page.getByText("Type the code the image shows.")).toBeVisible();
  await userEvent.fill(code(), "K7Q2");
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect
    .element(page.getByText("Larkspur passed the Test and listed 40 items."))
    .toBeVisible();
  expect(image("Larkspur").query()).toBeNull();
  expect(
    server.writes.filter((write) => write.method === "TEST").map((write) => write.body),
  ).toEqual([{}, { captcha: { attempt: "attempt-1", answer: "K7Q2" } }]);
});

test("an expired captcha on the Indexer's page says so, and New captcha loads another from its home", async () => {
  const server = indexersServer({
    captchaTests: [needsCaptcha],
    expireFirst: true,
  });
  renderApp({ path: pageOf(larkspurId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await userEvent.fill(code(), "LATE");
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect
    .element(
      page.getByRole("alert").getByText("The captcha expired. Load a new one and type its code."),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "New captcha" }));
  await expect.element(image("Larkspur")).toHaveAttribute("src", captchas[1]?.image);
  await userEvent.fill(code(), "K7Q2");
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect
    .element(page.getByText("Larkspur passed the Test and listed 40 items."))
    .toBeVisible();
  expect(server.writes.map((write) => write.method)).toEqual(["TEST", "TEST", "CAPTCHA", "TEST"]);
});

test("moving an Indexer whose session is a pasted cookie asks for a cookie for the new home", async () => {
  const server = indexersServer({ indexers: [pastedTorrentLeech, prowlarr, larkspur] });
  renderApp({ path: pageOf(torrentLeechId), handlers: server.handlers });
  await moveHome();
  await expect
    .element(
      dialog().getByText(
        "The browser cookie pasted for TorrentLeech works only from the Node it was pasted for. Paste the cookie of a browser signed in to TorrentLeech on home's network.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog().getByRole("button", { name: "Move" }));
  await expect
    .element(dialog().getByText("Paste the cookie of a browser signed in to TorrentLeech."))
    .toBeVisible();
  expect(server.writes).toEqual([]);

  await userEvent.fill(dialog().getByRole("textbox", { name: "Cookie" }), "session=abc");
  await userEvent.fill(dialog().getByRole("textbox", { name: "User-Agent" }), "Firefox/140.0");
  await userEvent.click(dialog().getByRole("button", { name: "Move" }));
  await expect.element(page.getByText("TorrentLeech searches from home now")).toBeVisible();
  expect(server.writes).toEqual([
    {
      method: "PATCH",
      id: torrentLeechId,
      body: {
        homeNodeId: homeId,
        cookie: { cookie: "session=abc", userAgent: "Firefox/140.0" },
      },
    },
  ]);
});

test("a move whose Test meets a captcha on the new home shows it, and Move answers its code", async () => {
  const sentence = "TorrentLeech needs a captcha to sign in. Type the code and test again.";
  const server = indexersServer({ captchaTests: [sentence] });
  renderApp({ path: pageOf(torrentLeechId), handlers: server.handlers });
  await moveHome();
  await userEvent.click(dialog().getByRole("button", { name: "Move" }));
  await expect.element(dialog().getByRole("alert").getByText(sentence)).toBeVisible();
  await expect.element(image("TorrentLeech")).toHaveAttribute("src", captchas[0]?.image);
  await userEvent.fill(code(), "K7Q2");
  await userEvent.click(dialog().getByRole("button", { name: "Move" }));
  await expect.element(page.getByText("TorrentLeech searches from home now")).toBeVisible();
  expect(server.writes.map((write) => write.body)).toEqual([
    { homeNodeId: homeId },
    { homeNodeId: homeId, captcha: { attempt: "attempt-1", answer: "K7Q2" } },
  ]);
});
