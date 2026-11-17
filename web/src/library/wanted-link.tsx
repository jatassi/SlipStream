import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import type { Noun } from "../format";
import { PendingLink } from "../imports/pending-link";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";
import { CalendarEntry } from "./calendar-entry";
import type { LibraryWork } from "./rows";
import { worksNounOf, worksWords } from "../wanted/wanted-words";

const failedGrabs = { one: "a failed Grab", many: "failed Grabs" } as const;

/**
 * "1,204 films need files or upgrades. 12 have failed Grabs." for the Works
 * Wanted lists by default, which leave out File gone, counted in `noun`.
 */
export function wantedWords(works: readonly LibraryWork[], noun: Noun): string | undefined {
  const listed = works.filter(({ wanted }) => wanted !== null && wanted !== "gone");
  if (listed.length === 0) {
    return undefined;
  }
  const failed = listed.filter(({ wanted }) => wanted === "failed").length;
  const all = `${worksWords(listed.length, noun)} ${listed.length === 1 ? "needs" : "need"} files or upgrades.`;
  return failed === 0
    ? all
    : `${all} ${String(failed)} ${failed === 1 ? "has" : "have"} ${failedGrabs[failed === 1 ? "one" : "many"]}.`;
}

/** The Library's row to Wanted, for this Library's Content Types, while any of its Works is wanted. */
function WantedLink({ types, works }: Scoped) {
  const [only] = types;
  const words = wantedWords(works, worksNounOf(types.length === 1 ? only : undefined));
  if (words === undefined) {
    return null;
  }
  return (
    <div className="mb-4">
      <List label="Wanted">
        <ListRow
          title={words}
          meta="What SlipStream searches for, and when it searches next."
          trailing={<Icon icon={ArrowRight01Icon} className="size-4" />}
          render={
            types.length === 1 && only !== undefined ? (
              <Link to="/library/wanted/$contentType" params={{ contentType: only.id }} />
            ) : (
              <Link to="/library/wanted" />
            )
          }
        />
      </List>
    </div>
  );
}

type Scoped = { types: readonly WorksContentType[]; works: readonly LibraryWork[] };

/** The rows above the Library to the pages under it: Pending imports, Wanted, then the Calendar. */
export function LibraryLinks(scoped: Scoped) {
  return (
    <>
      <PendingLink />
      <WantedLink {...scoped} />
      <CalendarEntry types={scoped.types} />
    </>
  );
}
