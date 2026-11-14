/** The last part of a path, whichever separator the server or root uses. */
export const fileName = (path: string): string => path.split(/[/\\]/u).findLast(Boolean) ?? path;
