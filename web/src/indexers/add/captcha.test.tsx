import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addIndexerServer } from "../../test/add-indexer-server";
import { captchas } from "../../test/catalog";
import { renderApp } from "../../test/render-app";

const larkspurForm = "/settings/indexers/new/catalog/larkspur";
const image = () => page.getByRole("img", { name: "The captcha Larkspur shows" });
const code = () => page.getByRole("textbox", { name: "Code" });
const attempts = (writes: ReturnType<typeof addIndexerServer>["writes"]) =>
  writes.filter((write) => write.method === "POST").map((write) => write.body["captcha"]);

async function signIn() {
  await userEvent.fill(page.getByRole("textbox", { name: "Username" }), "ada");
  await userEvent.fill(page.getByLabelText("Password", { exact: true }), "hunter2");
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
}

test("Continue loads the captcha home fetched, and the code is required", async () => {
  renderApp({ path: larkspurForm, handlers: addIndexerServer().handlers });
  expect(image().query()).toBeNull();
  await signIn();
  await expect.element(image()).toHaveAttribute("src", captchas[0]?.image);
  await expect
    .element(
      page.getByText("Signing in to Larkspur needs the code in this image, which home fetched.", {
        exact: false,
      }),
    )
    .toBeVisible();
  await expect.element(code()).toHaveFocus();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Type the code the image shows.")).toBeVisible();
});

test("a failed Test shows the site's words and a new image, whose code the next save answers", async () => {
  const sentence =
    "Larkspur refused the sign-in with 'Wrong code'. Check the username, password and any code, then test again.";
  const server = addIndexerServer({ failSaves: [sentence] });
  renderApp({ path: larkspurForm, handlers: server.handlers });
  await signIn();
  await userEvent.fill(code(), "WRONG");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert").getByText(sentence)).toBeVisible();
  await expect.element(image()).toHaveAttribute("src", captchas[1]?.image);
  await expect.element(code()).toHaveValue("");
  await userEvent.fill(code(), "K7Q2");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("heading", { name: "Larkspur", exact: true, level: 1 }))
    .toBeVisible();
  expect(attempts(server.writes)).toEqual([
    { attempt: "attempt-1", answer: "WRONG" },
    { attempt: "attempt-2", answer: "K7Q2" },
  ]);
});

test("an expired captcha says so, and New captcha loads another", async () => {
  const server = addIndexerServer({ expireFirst: true });
  renderApp({ path: larkspurForm, handlers: server.handlers });
  await signIn();
  await userEvent.fill(code(), "K7Q2");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(
      page.getByRole("alert").getByText("The captcha expired. Load a new one and type its code."),
    )
    .toBeVisible();
  expect(image().query()).toBeNull();
  await userEvent.click(page.getByRole("button", { name: "New captcha" }));
  await expect.element(image()).toHaveAttribute("src", captchas[1]?.image);
  await userEvent.fill(code(), "M3P9");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("heading", { name: "Larkspur", exact: true, level: 1 }))
    .toBeVisible();
  expect(attempts(server.writes)).toEqual([
    { attempt: "attempt-1", answer: "K7Q2" },
    { attempt: "attempt-2", answer: "M3P9" },
  ]);
});

test("a captcha is for one username: changing it asks for a new one", async () => {
  renderApp({ path: larkspurForm, handlers: addIndexerServer().handlers });
  await signIn();
  await expect.element(image()).toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Username" }), "grace");
  await expect.element(page.getByRole("button", { name: "Continue" })).toBeVisible();
  expect(image().query()).toBeNull();
});
