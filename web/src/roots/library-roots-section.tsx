import { Add01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListRootsSuspense, useListVolumesSuspense } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { ContentTypeSwitches } from "../works/content-type-switches";
import { type Acting, ActingDialogs } from "./acting-dialogs";
import { RootsGroup, groupsOf } from "./roots-group";
// oxlint-disable-next-line import/max-dependencies -- the page holds every block of Settings > Library roots
import { StorageSettings } from "../health/storage-settings";
import { MovesSettings } from "../moves/settings-section";
import { StorageBlock } from "./storage-block";

function Settings() {
  return (
    <>
      <StorageSettings toLibraryRoots={false} />
      <MovesSettings />
    </>
  );
}

export function LibraryRootsSection() {
  const { roots } = useListRootsSuspense().data;
  const volumes = useListVolumesSuspense().data.nodes;
  const { contentTypes } = useListContentTypesSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const [acting, setActing] = useState<Acting>();
  const groups = groupsOf(roots, contentTypes);
  const addFor = (groups.find((group) => group.roots.length === 0) ?? groups[0])?.type.id;
  return (
    <div className="flex flex-col gap-4">
      {contentTypes.length > 1 && (
        <Section title="Kinds of media">
          <ContentTypeSwitches />
        </Section>
      )}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-ink-2">
          A library root is a folder on a Storage Node where SlipStream keeps one kind of media,
          such as movies. New movies or series go into the default root for their kind.
        </p>
        <Button
          icon={Add01Icon}
          onClick={() => {
            setActing({ kind: "add", contentType: addFor ?? "" });
          }}
        >
          Add a library root
        </Button>
      </div>
      {groups.map((group) => (
        <RootsGroup
          key={group.type.id}
          group={group}
          nodes={nodes}
          volumes={volumes}
          onAct={setActing}
        />
      ))}
      <StorageBlock volumes={volumes} roots={roots} nodes={nodes} />
      <Settings />
      <ActingDialogs acting={acting} roots={roots} onAct={setActing} />
    </div>
  );
}
