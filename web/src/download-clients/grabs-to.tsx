import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { DownloadclientsClientCard as Card } from "../api/gen/model";
import { setDownloadClientSettings } from "../api/gen/query/downloadclients/downloadclients";
import { adoptGrabsTo } from "../live/adopt";
import { Select } from "../ui/select";
import { Refusal } from "../ui/setting-parts";
import { toast } from "../ui/toast";

const builtIn = "built-in";

/** "New Grabs go to": the Node's built-in client or one of its external clients that is on. */
export function GrabsTo({ card }: { card: Card }) {
  const queryClient = useQueryClient();
  const [refusal, setRefusal] = useState<string>();
  const options = [
    { value: builtIn, label: "Built-in client" },
    ...card.externals
      .filter((client) => client.on)
      .map((client) => ({ value: client.id, label: client.name })),
  ];
  const choose = async (value: string) => {
    setRefusal(undefined);
    try {
      const settings = await setDownloadClientSettings(card.nodeId, {
        grabsTo: value === builtIn ? null : value,
      });
      adoptGrabsTo(queryClient, card.nodeId, settings);
    } catch (error) {
      setRefusal(sentenceOf(error));
      return;
    }
    toast(
      `New Grabs on ${card.nodeName} go to ${options.find((option) => option.value === value)?.label}`,
    );
  };
  return (
    <div className="flex w-full max-w-80 flex-col gap-1.5">
      <Select
        label="New Grabs go to"
        options={options}
        value={card.grabsTo ?? builtIn}
        onValueChange={(value) => void choose(value)}
      />
      {refusal === undefined ? (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          Grabs for Works whose library root is on {card.nodeName} go to this client.
        </p>
      ) : (
        <Refusal error={refusal} />
      )}
    </div>
  );
}
