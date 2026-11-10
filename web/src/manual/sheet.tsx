import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import { sentenceOf } from "../api/errors";
import type { GrabsPinned } from "../api/gen/model";
import { getGetManualSearchQueryKey, getManualSearch } from "../api/gen/query/grabs/grabs";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { useShell } from "../ui/shell-context";
// oxlint-disable-next-line import/max-dependencies -- the sheet starts, reads and frames one Manual search on either shell
import { Sheet } from "../ui/sheet";
import { Skeleton } from "../ui/skeleton";
import { ManualBody } from "./body";
import { type ManualOpening, closeManualSearch, useManualSheet } from "./sheet-state";

const reading = "Loading the Manual search";

function Refused({ sentence }: { sentence: string }) {
  return (
    <div className="flex flex-col gap-4">
      <AlertLine text={sentence} />
      <DialogActions>
        <Button onClick={closeManualSearch}>Close</Button>
      </DialogActions>
    </div>
  );
}

/**
 * The Work's Manual search worked out around the pins. A swap keeps the old
 * plan on screen while the new one loads, and a closed sheet keeps nothing,
 * so the next opening reads the search it started.
 */
function useManualSearchRead(opening: ManualOpening) {
  const [pins, setPins] = useState<readonly GrabsPinned[]>([]);
  const params = pins.length === 0 ? undefined : { pins: pins.map(({ pin }) => pin) };
  const search = useQuery({
    queryKey: getGetManualSearchQueryKey(opening.workId, params),
    queryFn: ({ signal }) => getManualSearch(opening.workId, params, { signal }),
    enabled: opening.start.state === "started",
    placeholderData: keepPreviousData,
    gcTime: 0,
  });
  return { search, setPins };
}

function Opened({
  opening,
  read: { search, setPins },
}: {
  opening: ManualOpening;
  read: ReturnType<typeof useManualSearchRead>;
}) {
  if (opening.start.state === "refused") {
    return <Refused sentence={opening.start.sentence} />;
  }
  if (search.error !== null) {
    return <Refused sentence={sentenceOf(search.error)} />;
  }
  if (search.data === undefined) {
    return <Skeleton label={reading} />;
  }
  return (
    <ManualBody
      key={search.data.id}
      workId={opening.workId}
      scope={opening.scope}
      search={search.data}
      onPins={setPins}
      onReopen={() => {
        void search.refetch();
      }}
      onClose={closeManualSearch}
    />
  );
}

const onOpenChange = (next: boolean) => {
  if (!next) {
    closeManualSearch();
  }
};

function ManualSheet({ opening, open }: { opening: ManualOpening; open: boolean }) {
  const shell = useShell();
  const read = useManualSearchRead(opening);
  const title = read.search.data?.title ?? "Manual search";
  const body = <Opened opening={opening} read={read} />;
  return shell === "phone" ? (
    <Sheet title={title} fill open={open} onOpenChange={onOpenChange}>
      {body}
    </Sheet>
  ) : (
    <Dialog title={title} size={940} open={open} onOpenChange={onOpenChange}>
      {body}
    </Dialog>
  );
}

/**
 * Manual search: a plan in plain words to grab from, the Releases SlipStream
 * wouldn't grab, and every Release under All Releases. A 940 dialog on
 * desktop and a full sheet on phone, titled with the scope in words.
 */
export function ManualSearchHost() {
  const state = useManualSheet();
  // A sheet left open when the session ends must not greet the next one.
  useEffect(() => closeManualSearch, []);
  return state.shown === "none" ? null : (
    <ManualSheet key={state.opening.serial} opening={state.opening} open={state.shown === "open"} />
  );
}
