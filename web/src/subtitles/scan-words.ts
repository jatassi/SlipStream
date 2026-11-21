const counted = new Intl.NumberFormat("en");

/** A Storage Node's audio scans still to run, as the Subtitles section's lead ends. */
export const scansLeft = (node: string, left: number) =>
  `${node} has ${counted.format(left)} ${left === 1 ? "file" : "files"} left to scan for speech.`;

/** How many of the files a Profile change searches for wait on an audio scan. */
export const scansFirst = (count: number) =>
  count === 1
    ? "1 of them needs its audio scanned first."
    : `${counted.format(count)} of them need their audio scanned first.`;

/** A wanted subtitle that waits on its file's audio scan. */
export const waitsForScan = "Searching starts once SlipStream has scanned this file's audio.";
