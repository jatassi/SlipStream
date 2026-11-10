import { useQuery } from "@tanstack/react-query";
import { CancelCircleIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { GrabsFolded, GrabsManualSearch, GrabsPinned } from "../api/gen/model";
import {
  getListDownloadClientsQueryKey,
  listDownloadClients,
} from "../api/gen/query/downloadclients/downloadclients";
import { clientsOf } from "../grabs/send-to-sheet";
import { Details } from "../ui/details";
import { Icon } from "../ui/icon";
import { AllReleases } from "./all-releases";
import { GrabBar } from "./grab-bar";
// oxlint-disable-next-line import/max-dependencies -- the plan reads the clients, grabs, and draws each part of the sheet below the headline
import { useGrabbing } from "./grabbing";
import { GapRows, PickRows, type RowActs } from "./plan-rows";
import {
  type Choices,
  type PickRow,
  bodyOf,
  clientOf,
  hasPick,
  noChoices,
  pickOf,
  resultKey,
  sends,
  swap,
} from "./picks";

/** Red only for a fault in the Releases; a refusal reads in plain ink. */
function Folded({ folded }: { folded: GrabsFolded }) {
  return (
    <Details summary={folded.line}>
      <ul aria-label="Releases SlipStream wouldn't grab" className="flex flex-col gap-1">
        {folded.groups.map((group) => (
          <li
            key={group.words}
            className={`flex items-center gap-1.5 ${group.fault ? "text-fail-ink" : ""}`}
          >
            {group.fault && (
              <>
                <Icon icon={CancelCircleIcon} className="size-4 shrink-0" />
                <span className="sr-only">{"Fault: "}</span>
              </>
            )}
            {group.words}
          </li>
        ))}
      </ul>
    </Details>
  );
}

const toggled = (keys: ReadonlySet<string>, key: string, on: boolean) => {
  const next = new Set(keys);
  if (on) {
    next.add(key);
  } else {
    next.delete(key);
  }
  return next;
};

function useActs(
  workId: string,
  search: GrabsManualSearch,
  onPins: (pins: readonly GrabsPinned[]) => void,
) {
  const [choices, setChoices] = useState<Choices>(noChoices);
  const cards = useQuery({
    queryKey: getListDownloadClientsQueryKey(),
    queryFn: ({ signal }) => listDownloadClients({ signal }),
  }).data?.cards;
  const clients = clientsOf(cards ?? []);
  const grabbing = useGrabbing(workId, search.id);
  const sent = (row: PickRow, cancelGrabId?: string) => ({
    key: resultKey(row),
    pick: pickOf(row.pick, bodyOf(clients, clientOf(choices, row.key)), cancelGrabId),
  });
  const acts: RowActs = {
    choices,
    results: grabbing.results,
    clients,
    busy: grabbing.busy,
    onTick: (key, ticked) => {
      setChoices((before) => ({ ...before, unticked: toggled(before.unticked, key, !ticked) }));
    },
    onClient: (key, client) => {
      setChoices((before) => ({ ...before, clients: new Map(before.clients).set(key, client) }));
    },
    onPin: (key, pin) => {
      onPins(swap(search.pins, key, pin));
    },
    onCancelHeld: (row, grabId) => {
      grabbing.post([sent(row, grabId)]);
    },
  };
  const ticked = search.plan.rows.filter((row) => sends(choices, grabbing.results, row));
  return {
    acts,
    grabbing,
    ticked: ticked.length,
    setSendAll: (sendAll: string) => {
      setChoices((before) => ({ ...before, sendAll }));
    },
    grabTicked: () => {
      grabbing.post(ticked.map((row) => sent(row)));
    },
  };
}

/**
 * The plan below its headline: the ticked Grabs with Send all to and Grab,
 * the cells a Grab downloads and those nothing fills, the folded line and
 * All Releases.
 */
export function Plan({
  workId,
  search,
  onPins,
  onReopen,
}: {
  workId: string;
  search: GrabsManualSearch;
  onPins: (pins: readonly GrabsPinned[]) => void;
  onReopen: () => void;
}) {
  const { acts, grabbing, ticked, setSendAll, grabTicked } = useActs(workId, search, onPins);
  const picks = search.plan.rows.filter(hasPick);
  const gaps = search.plan.rows.filter((row) => !hasPick(row));
  return (
    <>
      {picks.length > 0 && (
        <>
          <PickRows rows={picks} acts={acts} />
          <GrabBar
            count={ticked}
            busy={grabbing.busy}
            clients={acts.clients}
            sendAll={acts.choices.sendAll}
            onSendAll={setSendAll}
            onGrab={grabTicked}
            failure={grabbing.failure}
            replaced={grabbing.replaced}
            onReopen={() => {
              grabbing.reset();
              onReopen();
            }}
          />
        </>
      )}
      {search.plan.held.length > 0 && (
        <ul aria-label="Cells a Grab is downloading" className="flex flex-col gap-1 text-ink-2">
          {search.plan.held.map((cell) => (
            <li key={cell.grabId}>{cell.words}</li>
          ))}
        </ul>
      )}
      {gaps.length > 0 && <GapRows rows={gaps} acts={acts} />}
      <Folded folded={search.folded} />
      <AllReleases workId={workId} count={search.releaseCount} grab={grabbing} />
    </>
  );
}
