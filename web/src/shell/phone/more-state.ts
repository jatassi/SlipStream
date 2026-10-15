import { createFlag } from "../flag";

const more = createFlag();

export const setMoreOpen = more.set;

export const openMore = () => {
  more.set(true);
};

export const useMoreOpen = more.useValue;
