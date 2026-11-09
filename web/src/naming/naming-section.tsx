import { Link } from "@tanstack/react-router";
import { useRef } from "react";

import type { NamingNamingList, NamingTypeNaming, WorksContentType } from "../api/gen/model";
import { useListNamingSuspense } from "../api/gen/query/naming/naming";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { ReviewAllRenames } from "../renames/review-all";
import type { Draft } from "./card";
import { TesterSheet } from "./tester-sheet";
import { TypeCard } from "./type-card";

/** The enabled Content Types SlipStream names files for, each with its naming. */
export const namedTypes = (
  contentTypes: readonly WorksContentType[],
  { contentTypes: namings }: NamingNamingList,
): { type: WorksContentType; naming: NamingTypeNaming }[] =>
  namings.flatMap((naming) => {
    const type = contentTypes.find((each) => each.id === naming.contentType && each.enabled);
    return type === undefined ? [] : [{ type, naming }];
  });

/** One card per enabled Content Type, so an Admin compares the types' templates side by side. */
export function NamingSection() {
  const { contentTypes } = useListContentTypesSuspense().data;
  const cards = namedTypes(contentTypes, useListNamingSuspense().data);
  const drafts = useRef(new Map<string, Draft>());
  return cards.length === 0 ? (
    <p className="text-ink-2">
      Turn on a Content Type in{" "}
      <Link
        to="/settings/$section"
        params={{ section: "content-types" }}
        className="rounded-item font-medium text-ink-1 underline-offset-4 hover:underline"
      >
        Content Types
      </Link>{" "}
      to name its files.
    </p>
  ) : (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        <ReviewAllRenames />
        <TesterSheet
          types={contentTypes}
          draftOf={(type) => drafts.current.get(type) ?? new Map()}
        />
      </div>
      {cards.map(({ type, naming }) => (
        <TypeCard
          key={type.id}
          type={type}
          naming={naming}
          onDraft={(draft) => {
            drafts.current.set(type.id, draft);
          }}
        />
      ))}
    </div>
  );
}
