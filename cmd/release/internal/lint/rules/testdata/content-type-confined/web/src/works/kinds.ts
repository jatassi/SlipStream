// The "music" kind is a test type.
/* 'movie'
   "tv" */
export const kinds = ["Movie", 'tv', `music`, "musician"];
export const label = (kind: string) => `${kind}music`;
export const nested = `a${`movie`}b${{ "tv": 1 }["tv"]}`;
export const apostrophe = "it's"; const after = 'tv';
