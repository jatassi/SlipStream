import { z } from "zod";

import { isProblem, sentenceOf } from "../api/errors";
import type { RootsBusy, RootsInUse, RootsMissingFiles } from "../api/gen/model";
import {
  RootsBusy as BusyBody,
  RootsInUse as InUseBody,
  RootsMissingFiles as MissingBody,
  RootsRefusal as RefusalBody,
} from "../api/gen/zod/slipStream.schemas.zod";
import { type Namer, refusalSentence } from "./words";

const unusable = z.object({ refusal: RefusalBody });
const missing = z.object({ missing: MissingBody });
const inUse = z.object({ use: InUseBody });
const busy = z.object({ busy: BusyBody });

export function placeRefusal(error: unknown, namer: Namer): string {
  if (isProblem(error, "root_unusable")) {
    return refusalSentence(unusable.parse(error.body).refusal, namer);
  }
  if (isProblem(error, "node_offline")) {
    return `${namer.node} is offline, so it can't check the folder.`;
  }
  return sentenceOf(error);
}

/** The recorded files a re-home found missing at the new place, when that is why it refused. */
export const missingOf = (error: unknown): RootsMissingFiles | undefined =>
  isProblem(error, "rehome_missing") ? missing.parse(error.body).missing : undefined;

const counted = new Intl.NumberFormat("en");

export const filesWords = (count: number) =>
  count === 1 ? "1 file" : `${counted.format(count)} files`;

const inFlight = (count: number, noun: string) =>
  count === 1 ? `a ${noun}` : `${counted.format(count)} ${noun}s`;

function inUseWords({ files, grabs, moves }: RootsInUse): string {
  const parts = [
    files > 0 ? `holds ${filesWords(files)}` : "",
    grabs > 0 ? `has ${inFlight(grabs, "Grab")} in flight` : "",
    moves > 0 ? `has ${inFlight(moves, "Move")} in flight` : "",
  ].filter(Boolean);
  return listed(parts);
}

const listed = (parts: readonly string[]) =>
  parts.length > 1 ? `${parts.slice(0, -1).join(", ")} and ${parts.at(-1)}` : parts.join("");

const many = (count: number, noun: string) =>
  `${counted.format(count)} ${noun}${count === 1 ? "" : "s"}`;

function busyWords({ moves, imports, transfers }: RootsBusy): string {
  const using = [
    moves > 0 ? many(moves, "Move") : "",
    imports > 0 ? many(imports, "Import") : "",
    transfers > 0 ? many(transfers, "Transfer") : "",
  ].filter(Boolean);
  const verb = moves + imports + transfers === 1 ? "uses" : "use";
  const cancel = moves > 0 ? ", or cancel the Moves in Activity" : "";
  return `while ${listed(using)} ${verb} it. Wait for them to finish${cancel}.`;
}

/** Why a re-home refused while Moves, Imports or Transfers use the root, named by label. */
export const busyRefusal = (error: unknown, label: string): string | undefined =>
  isProblem(error, "root_busy")
    ? `The ${label} root can't be re-homed ${busyWords(busy.parse(error.body).busy)}`
    : undefined;

export function deleteRefusal(error: unknown): string {
  if (isProblem(error, "root_in_use")) {
    return `It can't be deleted while it ${inUseWords(inUse.parse(error.body).use)}.`;
  }
  return sentenceOf(error);
}
