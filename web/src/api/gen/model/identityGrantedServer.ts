export interface IdentityGrantedServer {
  /** The kind's product name, such as Plex */
  kind: string;
  name: string;
  /** Plex only: true adds the Member to the Plex Home; false shares the server as a friend */
  plexHome: boolean;
  /** False once the server is removed */
  present: boolean;
  server: string;
  /** Whether Show to Members is on. A server that is off isn't offered when the link is used */
  shown: boolean;
}
