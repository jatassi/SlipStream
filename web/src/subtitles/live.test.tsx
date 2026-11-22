import { expect, test } from "vitest";

import {
  getGetSubtitleAttemptQueryKey,
  getGetSubtitleNodeSettingsQueryKey,
  getGetSubtitleSettingsQueryKey,
  getGetWorkSubtitlesQueryKey,
  getListSubtitleProvidersQueryKey,
} from "../api/gen/query/subtitles/subtitles";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const attic = "4b6d8f0a-2c4e-4a6c-8e0a-2c4e6a8c0e2a";
const cellar = "6d8f0a2c-4e6a-4c8e-8a2c-4e6a8c0e2a4c";
const film = "8f0a2c4e-6a8c-4e0a-8c4e-6a8c0e2a4c6d";
const tried = "0a2c4e6a-8c0e-4a2c-8e6a-8c0e2a4c6d8f";

test("subtitles.changed refreshes the named Works' subtitles and the section's lead", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/library", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    attic: getGetWorkSubtitlesQueryKey(attic),
    cellar: getGetWorkSubtitlesQueryKey(cellar),
    settings: getGetSubtitleSettingsQueryKey(),
    providers: getListSubtitleProvidersQueryKey(),
  });
  server.clients[0]?.send({
    event: "subtitles.changed",
    data: { workIds: [attic], fileIds: [film] },
  });
  await expect
    .poll(invalidated)
    .toEqual({ attic: true, cellar: false, settings: true, providers: false });
});

test("subtitles.providers-changed refreshes the providers and every subtitle setting", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/library", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    providers: getListSubtitleProvidersQueryKey(),
    settings: getGetSubtitleSettingsQueryKey(),
    node: getGetSubtitleNodeSettingsQueryKey(cellar),
    attic: getGetWorkSubtitlesQueryKey(attic),
  });
  server.clients[0]?.send({ event: "subtitles.providers-changed", data: {} });
  await expect
    .poll(invalidated)
    .toEqual({ providers: true, settings: true, node: true, attic: false });
});

test("subtitles.attempt refreshes only that attempt", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/library", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    tried: getGetSubtitleAttemptQueryKey(film, tried),
    other: getGetSubtitleAttemptQueryKey(film, attic),
  });
  server.clients[0]?.send({ event: "subtitles.attempt", data: { attemptId: tried, fileId: film } });
  await expect.poll(invalidated).toEqual({ tried: true, other: false });
});
