import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import type { WorksContentType } from "../api/gen/model";
import { SegmentedLinks, segmentedLink } from "../ui/segmented-links";

const exact = { exact: true, includeSearch: false };

function TypeSwitch({ types }: { types: readonly WorksContentType[] }) {
  return (
    <SegmentedLinks label="Content Types">
      <Link to="/library" activeOptions={exact} className={segmentedLink}>
        All
      </Link>
      {types.map((type) => (
        <Link
          key={type.id}
          to="/library/$contentType"
          params={{ contentType: type.id }}
          activeOptions={exact}
          className={segmentedLink}
        >
          {type.name}
        </Link>
      ))}
    </SegmentedLinks>
  );
}

/**
 * The switch between "All" and each enabled type's Library, in the
 * sidebar's order. Each segment is its own address, so it opens on the view
 * that Library last used. A Library of one enabled type has no switch.
 */
export const typeSwitch = (enabled: readonly WorksContentType[]): ReactNode =>
  enabled.length < 2 ? undefined : <TypeSwitch types={enabled} />;
