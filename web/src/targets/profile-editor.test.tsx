import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsNewProfile, TargetsProfile, TargetsTarget } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { CreateProfileBody } from "../api/gen/zod/targets/targets.zod";
import { cinema } from "../test/library";
import { films, profileHandlers, ultra } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { keeper, starter, targetHandlers } from "../test/targets";

function creating() {
  const sent: TargetsNewProfile[] = [];
  const handler = http.post("*/api/v1/profiles", async ({ request }) => {
    const body = CreateProfileBody.parse(await request.json());
    sent.push(body);
    const made: TargetsProfile = {
      ...ultra,
      id: "0199e1f2-7a3b-7c4d-8e5f-00000000000a",
      name: body.name,
    };
    return HttpResponse.json(made, { status: 201 });
  });
  return { sent, handler };
}

const handlers = (
  targets: readonly TargetsTarget[] = [starter, keeper],
  profiles: readonly TargetsProfile[] = [films, ultra],
) => [
  ...targetHandlers(targets),
  ...profileHandlers(profiles),
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
];

const rowsOf = (list: string) =>
  page
    .getByRole("list", { name: list, exact: true })
    .getByRole("listitem")
    .all()
    .map((row) => row.element().textContent);

const sentence = () => page.getByText(/^Every Item |^Each Item /u);

async function chooseCuts() {
  await userEvent.click(page.getByRole("radio", { name: "Choose cuts" }));
  await expect.element(sentence()).toHaveTextContent("Every Item always wants Standard.");
}

test("a new Profile takes Targets in an order the keyboard sets, and creates with Any cut", async () => {
  const { sent, handler } = creating();
  const { router } = renderApp({
    path: "/settings/profiles/new",
    handlers: [...handlers(), handler],
  });
  await expect.element(page.getByRole("heading", { name: "New Profile", level: 1 })).toBeVisible();
  await expect
    .element(page.getByText("This Profile wants nothing.", { exact: false }))
    .toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Both");
  for (const name of ["1080p", "2160p HDR"]) {
    // oxlint-disable-next-line no-await-in-loop -- each pick lands before the next, as a person's does
    await userEvent.click(page.getByRole("button", { name: "Add Target" }));
    // oxlint-disable-next-line no-await-in-loop -- as above
    await userEvent.click(page.getByRole("menuitem", { name }));
  }
  await expect.element(page.getByRole("button", { name: "Add Target" })).toBeDisabled();
  expect(rowsOf("Targets")).toEqual([
    "Rank 11080pNo estimate yet",
    "Rank 22160p HDRNo estimate yet",
  ]);

  page.getByRole("button", { name: "Move 2160p HDR", exact: true }).element().focus();
  await userEvent.keyboard("{ArrowUp}");
  await expect.element(page.getByText("2160p HDR moved to rank 1 of 2.")).toBeInTheDocument();
  expect(rowsOf("Targets")).toEqual([
    "Rank 12160p HDRNo estimate yet",
    "Rank 21080pNo estimate yet",
  ]);

  await userEvent.click(page.getByRole("button", { name: "Create Profile" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/profiles");
  expect(sent).toEqual([
    {
      name: "Both",
      contentType: "cinema",
      targets: [
        { targetId: keeper.id, sourcing: "download" },
        { targetId: starter.id, sourcing: "download" },
      ],
      editions: { kind: "any" },
    },
  ]);
});

test("Choose cuts lists cuts from Standard, says what they want, and warns once Standard is gone", async () => {
  renderApp({ path: "/settings/profiles/new", handlers: handlers() });
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Each Item wants one file at every Target, of whatever cut. A file's cut only sets its name.",
    );
  await chooseCuts();
  await expect.element(page.getByRole("button", { name: "Remove Standard" })).toBeDisabled();

  await userEvent.click(page.getByRole("button", { name: "Add cut" }));
  const offered = page
    .getByRole("menuitem")
    .all()
    .map((item) => item.element().textContent);
  expect(offered).toEqual(["Theatrical", "Extended", "Director's Cut", "IMAX", "Superfan"]);
  await userEvent.click(page.getByRole("menuitem", { name: "Extended" }));
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Every Item always wants Standard. It wants Extended only once a release or file of that cut has been seen for it.",
    );

  await userEvent.click(page.getByRole("button", { name: "Remove Standard" }));
  expect(rowsOf("Cuts")).toEqual(["Rank 1Extended"]);
  await expect.element(page.getByText("No Standard")).toBeVisible();
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Every Item wants Extended outright, even where no release of it has been seen. A release that names no cut is Standard, which this list doesn't want.",
    );
  await expect.element(page.getByRole("button", { name: "Remove Extended" })).toBeDisabled();

  await userEvent.click(page.getByRole("radio", { name: "Any cut" }));
  await expect.element(page.getByText("No Standard")).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("radio", { name: "Choose cuts" }));
  expect(rowsOf("Cuts")).toEqual(["Rank 1Extended"]);
});

test("a saved Profile shows each Target's disk estimate, flags a contained Target, and fixes its type", async () => {
  const small: TargetsTarget = {
    ...keeper,
    id: "0199e1f2-7a3b-7c4d-8e5f-6a7b8c9d0e5d",
    name: "1080p small",
    tiers: [["webdl-1080p"]],
    cutoff: "webdl-1080p",
  };
  const holding = { ...starter, contains: [small.id] };
  const both: TargetsProfile = {
    ...ultra,
    targets: [...ultra.targets, { targetId: small.id, sourcing: "download" }],
  };
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: handlers([holding, keeper, small], [films, both]),
  });
  await expect.element(page.getByRole("heading", { name: "Ultra", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Name" })).toHaveValue("Ultra");
  await expect
    .element(page.getByText("For Films. A Profile's Content Type is fixed."))
    .toBeVisible();
  expect(rowsOf("Targets")).toEqual([
    "Rank 12160p HDRNo estimate yet",
    "Rank 21080pAbout 2.1 GB across its Works, at 20 MiB per minute",
    "Rank 31080p smallNo estimate yetEvery file 1080p small takes also meets 1080p.",
  ]);
  await expect
    .element(page.getByRole("button", { name: /^(Create|Save)/u }))
    .not.toBeInTheDocument();
});

test("removing every Target leaves a Profile that wants nothing", async () => {
  renderApp({ path: `/settings/profiles/${films.id}`, handlers: handlers() });
  await userEvent.click(page.getByRole("button", { name: "Remove 1080p" }));
  await expect
    .element(
      page.getByText(
        "This Profile wants nothing. Add a Target to say which versions its Works keep.",
      ),
    )
    .toBeVisible();
});

test("creating with chosen cuts is held back with the reason, and sends nothing", async () => {
  const { sent, handler } = creating();
  renderApp({ path: "/settings/profiles/new", handlers: [...handlers(), handler] });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Cuts");
  await chooseCuts();
  await userEvent.click(page.getByRole("button", { name: "Create Profile" }));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("Chosen cuts can't be saved yet. Choose Any cut to create the Profile.");
  expect(sent).toEqual([]);
});
