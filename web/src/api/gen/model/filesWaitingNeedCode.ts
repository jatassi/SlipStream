/**
 * Why the Grab waits: node_away while its Node is out of step, client_away while its Torrent's client isn't responding, refused the sign-in or needs attention
 */
export type FilesWaitingNeedCode = "node_away" | "client_away";
