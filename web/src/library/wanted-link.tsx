import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import { PendingLink } from "../imports/pending-link";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";
import type { LibraryWork } from "./rows";

const counted = new Intl.NumberFormat("en");

/** "1,204 wanted. 12 failed." for the Works Wanted lists by default, which leave out File gone. */
export function wantedWords(works: readonly LibraryWork[]): string | undefined {
  const listed = works.filter(({ wanted }) => wanted !== null && wanted !== "gone");
  if (listed.length === 0) {
    return undefined;
  }
  const failed = listed.filter(({ wanted }) => wanted === "failed").length;
  const all = `${counted.format(listed.length)} wanted.`;
  return failed === 0 ? all : `${all} ${counted.format(failed)} failed.`;
}

/** The Library's row to Wanted, for this Library's Content Types, while any of its Works is wanted. */
function WantedLink({ types, works }: Scoped) {
  const words = wantedWords(works);
  if (words === undefined) {
    return null;
  }
  const [only] = types;
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

/** The rows above the Library to the pages under it: Pending imports, then Wanted. */
export function LibraryLinks(scoped: Scoped) {
  return (
    <>
      <PendingLink />
      <WantedLink {...scoped} />
    </>
  );
}
