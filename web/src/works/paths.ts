const lastSeparator = (path: string) => Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\"));

export const nameOf = (path: string): string => path.slice(lastSeparator(path) + 1);

export const folderOf = (path: string): string => path.slice(0, lastSeparator(path));
