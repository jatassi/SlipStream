import { HttpResponse, http } from "msw";
import { z } from "zod";

import type {
  NotificationsConnection,
  NotificationsFeed,
  NotificationsFeedRow,
} from "../api/gen/model";
import { problem } from "./handlers";
import { home } from "./nodes";
import { events, family, notificationBell, notificationSettings, providers } from "./notifications";

const api = "/api/v1/notifications";

export type NotificationWrite = { readonly path: string; readonly body: unknown };

type Write = (path: string, request: Request) => Promise<Record<string, unknown>>;

type TestFails = { detail: string; field: string };

function tested(fails: TestFails | undefined) {
  return fails === undefined
    ? HttpResponse.json({ node: home.name })
    : HttpResponse.json(
        {
          status: 422,
          code: "notification_test_failed",
          title: "The Test failed",
          detail: fails.detail,
          field: fails.field,
        },
        { status: 422, headers: { "Content-Type": "application/problem+json" } },
      );
}

const readHandlers = (list: NotificationsConnection[]) => [
  http.get(`*${api}/providers`, () => HttpResponse.json({ providers })),
  http.get(`*${api}/events`, () => HttpResponse.json({ events })),
  http.get(`*${api}/connections`, () => HttpResponse.json({ connections: list })),
  http.get(`*${api}/bell`, () => HttpResponse.json(notificationBell)),
  http.get(`*${api}/settings`, () => HttpResponse.json({ settings: notificationSettings() })),
];

const testHandlers = (write: Write, fails: TestFails | undefined) => [
  http.post(`*${api}/connections/test`, async ({ request }) => {
    await write("test-draft", request);
    return tested(fails);
  }),
  http.post(`*${api}/connections/:id/test`, async ({ request, params }) => {
    await write(`test ${String(params["id"])}`, request);
    return tested(fails);
  }),
];

const saveHandlers = (list: NotificationsConnection[], write: Write) => [
  http.post(`*${api}/connections`, async ({ request }) => {
    const body = await write("add", request);
    const made = {
      ...family,
      id: "0199f3c0-3333-7000-8000-0000000000aa",
      name: String(body["name"]),
    };
    list.push(made);
    return HttpResponse.json(made);
  }),
  http.patch(`*${api}/connections/:id`, async ({ request, params }) => {
    const body = await write(`edit ${String(params["id"])}`, request);
    const at = list.findIndex(({ id }) => id === params["id"]);
    const was = list[at];
    if (was === undefined) {
      return problem(
        404,
        "notification_connection_not_found",
        "No notification connection has this id",
      );
    }
    const got = { ...was, ...(typeof body["name"] === "string" && { name: body["name"] }) };
    list[at] = got;
    return HttpResponse.json(got);
  }),
];

/** The feed newest first, limit rows at a time below before, and a seen post that zeroes unread. */
function feedHandlers(feed: readonly NotificationsFeedRow[], unreadAtStart: number, write: Write) {
  let unread = unreadAtStart;
  return [
    http.get(`*${api}/feed`, ({ request }) => {
      const url = new URL(request.url);
      const before = Number(url.searchParams.get("before") ?? Infinity);
      const limit = Number(url.searchParams.get("limit") ?? 50);
      const below = feed.filter(({ seq }) => seq < before);
      const rows = below.slice(0, limit);
      const last = rows.at(-1);
      const page: NotificationsFeed = {
        rows,
        unread,
        before: below.length > limit && last !== undefined ? last.seq : null,
      };
      return HttpResponse.json(page);
    }),
    http.post(`*${api}/feed/seen`, async ({ request }) => {
      await write("seen", request);
      unread = 0;
      return new HttpResponse(null, { status: 204 });
    }),
  ];
}

/**
 * A Notifications API that keeps what the section and the bell write: Test
 * answers with home or the failure the test names, Save answers with the
 * connection as edited, and the feed pages its rows newest first.
 */
export function notificationsWorld({
  start = [family],
  testFails,
  feed = [],
  unread = 0,
}: {
  start?: NotificationsConnection[];
  testFails?: TestFails;
  feed?: NotificationsFeedRow[];
  unread?: number;
} = {}) {
  const list = [...start];
  const writes: NotificationWrite[] = [];
  const write: Write = async (path, request) => {
    const text = await request.text();
    const body = z.record(z.string(), z.unknown()).parse(text === "" ? {} : JSON.parse(text));
    writes.push({ path, body });
    return body;
  };
  const handlers = [
    ...readHandlers(list),
    ...testHandlers(write, testFails),
    ...saveHandlers(list, write),
    ...feedHandlers(feed, unread, write),
  ];
  return { handlers, writes, list };
}
