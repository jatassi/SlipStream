import { HttpResponse, http } from "msw";

import type {
  SettingField,
  WorksMonitoringChange,
  WorksOptionsChange,
  WorksOptionsChanged,
  WorksPresetCounts,
} from "../api/gen/model";
import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../api/gen/query/roots/roots.msw";
import {
  getListContentTypesMockHandler,
  getSetWorksMonitoringMockHandler,
} from "../api/gen/query/works/works.msw";
import { SetWorkOptionsBody, SetWorksMonitoringBody } from "../api/gen/zod/works/works.zod";
import { anthologies, longShelf } from "./anthology";
import { problem } from "./handlers";
import { cinema, libraryRow, libraryServer, orchardRow } from "./library";
import { profileHandlers } from "./profiles";
import { filmsRoot, mediaVolumes } from "./roots";

/** What each preset monitors in The Long Shelf: 24 Volumes of 49 Stories, and 24 Extras. */
export const longShelfCounts: WorksPresetCounts = {
  asides: 24,
  choices: [
    { preset: "all", monitors: 1176, monitorsNew: true },
    { preset: "future", monitors: 0, monitorsNew: true },
    { preset: "missing", monitors: 1176, monitorsNew: true },
    { preset: "existing", monitors: 0, monitorsNew: false },
    { preset: "first", monitors: 49, monitorsNew: false },
    { preset: "latest", monitors: 49, monitorsNew: true },
    { preset: "none", monitors: 0, monitorsNew: false },
  ],
};

type Moves = { readonly start: number; readonly stop: number };

/**
 * PUT /works/monitoring over Works that each count as The Long Shelf does,
 * each moving what `moves` says for the body. `sent` records every body.
 */
export function monitoringServer(moves: (body: WorksMonitoringChange) => Moves) {
  const sent: WorksMonitoringChange[] = [];
  return {
    sent,
    handler: getSetWorksMonitoringMockHandler(async ({ request }) => {
      const body = SetWorksMonitoringBody.parse(await request.json());
      sent.push(body);
      const { start, stop } = moves(body);
      return {
        outcomes: body.workIds.map((workId) => ({
          workId,
          outcome: "done" as const,
          start,
          stop,
          counts: longShelfCounts,
        })),
      };
    }),
  };
}

const withValue = (field: SettingField, value: unknown): SettingField => {
  if (field.kind === "choice" && typeof value === "string") {
    return { ...field, value, running: value, isDefault: value === field.default };
  }
  if (field.kind === "switch" && typeof value === "boolean") {
    return { ...field, value, running: value, isDefault: value === field.default };
  }
  return field;
};

/**
 * PUT /works/{id}/options for The Long Shelf: an order change moves
 * Stories and a preview of one withdraws `withdraws` of them, an order in
 * `unavailable` is refused, and so is any change to a key `refused` maps
 * to its sentence.
 */
export function optionsServer({
  withdraws,
  unavailable = [],
  refused = {},
}: {
  withdraws: number;
  unavailable?: readonly string[];
  refused?: Readonly<Record<string, string>>;
}) {
  const sent: WorksOptionsChange[] = [];
  return {
    sent,
    handler: http.put("*/api/v1/works/:workId/options", async ({ request }) => {
      const body = SetWorkOptionsBody.parse(await request.json());
      sent.push(body);
      const type = body.type ?? {};
      const order = type["order"];
      if (typeof order === "string" && unavailable.includes(order)) {
        return problem(
          422,
          "order_unavailable",
          "The metadata source doesn't list this Work in that order",
        );
      }
      const sentence = Object.keys(type)
        .map((key) => refused[key])
        .find((each) => each !== undefined);
      if (sentence !== undefined) {
        return problem(
          422,
          "setting_out_of_range",
          "The value is outside the setting's range",
          sentence,
        );
      }
      const changed: WorksOptionsChanged = {
        levelFolders: body.levelFolders ?? longShelf.levelFolders,
        options: longShelf.options.map((field) =>
          body.preview ? field : withValue(field, type[field.key]),
        ),
        moves: order !== undefined,
        withdraws: order === undefined ? 0 : withdraws,
      };
      return HttpResponse.json(changed);
    }),
  };
}

export const longShelfRow = libraryRow({
  id: longShelf.id,
  contentType: anthologies.id,
  key: longShelf.key,
  label: longShelf.label,
  items: 1200,
});

export const shortTalesRow = libraryRow({
  id: "6a8c0e2a-4c6e-4a8c-9e0a-2c4e6a8c0e2c",
  contentType: anthologies.id,
  key: "a2",
  label: "Short Tales",
  items: 300,
});

/** The two anthologies' rows beside the Quiet Orchard's, with the roots and Profiles the Library reads. */
export const mixedLibrary = [
  getListContentTypesMockHandler({ contentTypes: [cinema, anthologies] }),
  libraryServer([orchardRow, longShelfRow, shortTalesRow]).handler,
  ...profileHandlers(),
  getListRootsMockHandler({ roots: [filmsRoot] }),
  getListVolumesMockHandler({ nodes: [mediaVolumes] }),
];
