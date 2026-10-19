import type { RefObject } from "react";

import type { Browser } from "./use-browser";

export type Parts = {
  browser: Browser;
  field: RefObject<HTMLInputElement | null>;
  list: RefObject<HTMLDivElement | null>;
  listId: string;
  verdictId: string;
  heights: { readonly one: number; readonly two: number };
  nodeName: string;
  confirmVerb: string;
  allowNewFolder: boolean;
  close: () => void;
};
