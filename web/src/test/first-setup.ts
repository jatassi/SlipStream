import { HttpResponse, http, sse } from "msw";
import { page } from "vitest/browser";

import type {
  FirstrunFirstRun,
  FirstrunStepID,
  FirstrunStepState,
  MailSettings,
} from "../api/gen/model";
import { SetFirstRunStepBody } from "../api/gen/zod/firstrun/firstrun.zod";
import { type LiveMap, ada } from "./handlers";

type Client = Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"];

export const todo = (id: FirstrunStepID): FirstrunStepState => ({ id, state: "todo" });
export const done = (id: FirstrunStepID): FirstrunStepState => ({ id, state: "done" });
export const skipped = (id: FirstrunStepID): FirstrunStepState => ({ id, state: "skipped" });

const mailSet: MailSettings = {
  fromAddress: "slipstream@example.test",
  fromName: "SlipStream",
  host: "smtp.example.test",
  passwordSet: true,
  port: 587,
  security: "starttls",
  username: "slipstream",
};

/**
 * An Admin's open first setup on the Controller: the session says so, each
 * step write stores its outcome, and Finish skips every todo step. A test
 * changes `state` and sends firstrun.changed to play another tab's write.
 */
export function firstSetup(steps: readonly FirstrunStepState[]) {
  const state: FirstrunFirstRun = { open: true, steps: [...steps] };
  const writes: { id: string; outcome: string }[] = [];
  const finishes: number[] = [];
  const clients: Client[] = [];
  const handlers = [
    http.get("*/api/v1/session", () =>
      HttpResponse.json({
        account: ada,
        pinRequired: false,
        keyPrompt: "ask",
        firstRun: state.open ? "open" : "finished",
      }),
    ),
    http.get("*/api/v1/first-run", () => HttpResponse.json(state)),
    http.get("*/api/v1/mail/settings", () => HttpResponse.json(mailSet)),
    http.put("*/api/v1/first-run/steps/:id", async ({ params, request }) => {
      const id = String(params["id"]);
      const { outcome } = SetFirstRunStepBody.parse(await request.json());
      writes.push({ id, outcome });
      state.steps = state.steps.map((step) =>
        step.id === id ? { id: step.id, state: outcome } : step,
      );
      return HttpResponse.json(state);
    }),
    http.post("*/api/v1/first-run/finish", () => {
      finishes.push(Date.now());
      state.open = false;
      state.steps = state.steps.map((step) =>
        step.state === "todo" ? { id: step.id, state: "skipped" } : step,
      );
      return new HttpResponse(null, { status: 204 });
    }),
    sse<LiveMap>("*/api/v1/events", ({ client }) => {
      clients.push(client);
      client.send({ event: "live.resync", data: {} });
    }),
  ];
  return {
    handlers,
    state,
    writes,
    finishes,
    send: (message: Parameters<Client["send"]>[0]) => {
      for (const client of clients) {
        client.send(message);
      }
    },
  };
}

export const stepHeading = (name: string) => page.getByRole("heading", { name, level: 1 });

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const choice = (key: string, label: string, value: string, options: [string, string][]) => ({
  ...about,
  kind: "choice" as const,
  key,
  label,
  isDefault: value === options[0]?.[0],
  value,
  running: value,
  default: options[0]?.[0] ?? value,
  options: options.map(([option, name]) => ({ value: option, label: name })),
});

/** The metadata language and region, at `language` and `region`; the first option of each is the default. */
export function metadataPlace(language = "en-US", region = "US") {
  const sent: unknown[] = [];
  const settings = () => ({
    settings: [
      choice("metadata.language", "Metadata language", language, [
        ["en-US", "English (United States)"],
        ["en-GB", "English (United Kingdom)"],
        ["fr-FR", "French (France)"],
        ["de-DE", "German (Germany)"],
      ]),
      choice("metadata.region", "Metadata region", region, [
        ["US", "United States"],
        ["FR", "France"],
        ["DE", "Germany"],
        ["GB", "United Kingdom"],
      ]),
    ],
  });
  return {
    sent,
    handlers: [
      http.get("*/api/v1/metadata/settings", () => HttpResponse.json(settings())),
      http.patch("*/api/v1/metadata/settings", async ({ request }) => {
        sent.push(await request.json());
        return HttpResponse.json(settings());
      }),
    ],
  };
}
