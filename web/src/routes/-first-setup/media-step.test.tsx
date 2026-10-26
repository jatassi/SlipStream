import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksContentType } from "../../api/gen/model";
import { SetContentTypeEnabledBody } from "../../api/gen/zod/works/works.zod";
import { done, firstSetup, stepHeading, todo } from "../../test/first-setup";
import { problem } from "../../test/handlers";
import { cinema } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { zines } from "../../test/roots-page";

function contentTypes(start: WorksContentType[], refuse?: string) {
  let types = start;
  const sent: { id: string; enabled: boolean }[] = [];
  return {
    sent,
    handlers: [
      http.get("*/api/v1/content-types", () => HttpResponse.json({ contentTypes: types })),
      http.put("*/api/v1/content-types/:id/enabled", async ({ params, request }) => {
        const id = String(params["id"]);
        const { enabled } = SetContentTypeEnabledBody.parse(await request.json());
        sent.push({ id, enabled });
        if (id === refuse) {
          return problem(
            409,
            "content_type_in_use",
            "Remove its Works before turning this Content Type off",
          );
        }
        types = types.map((type) => (type.id === id ? { ...type, enabled } : type));
        return HttpResponse.json(types.find((type) => type.id === id));
      }),
    ],
  };
}

function open(types: ReturnType<typeof contentTypes>) {
  const server = firstSetup([todo("media"), done("roots")]);
  renderApp({
    path: "/setup/media",
    handlers: [
      ...server.handlers,
      ...types.handlers,
      http.get("*/api/v1/roots", () => HttpResponse.json({ roots: [] })),
    ],
  });
  return server;
}

test("the kinds of media step turns a Content Type off and keeps the last one on", async () => {
  const types = contentTypes([cinema, zines]);
  const server = open(types);
  await expect.element(stepHeading("Kinds of media")).toBeVisible();
  const films = page.getByRole("switch", { name: "Films" });
  await expect.element(films).toBeChecked();
  await userEvent.click(page.getByText("Zines"));
  await expect.element(page.getByRole("switch", { name: "Zines" })).not.toBeChecked();
  await expect.element(films).toBeDisabled();
  await expect.element(page.getByText("At least one kind of media stays on.")).toBeVisible();
  expect(types.sent).toEqual([{ id: "zine", enabled: false }]);
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("Library roots")).toBeVisible();
  expect(server.writes).toEqual([{ id: "media", outcome: "done" }]);
});

test("a Content Type that has Works stays on and says why on its switch", async () => {
  const types = contentTypes([cinema, zines], "zine");
  open(types);
  await userEvent.click(page.getByText("Zines"));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("Remove its Works before turning this Content Type off");
  await expect.element(page.getByRole("switch", { name: "Zines" })).toBeChecked();
});
