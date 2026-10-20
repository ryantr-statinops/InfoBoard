// Projection synchronization state machine for InfoBoard (IP-09).
// States: Disconnected -> Connecting -> Handshaking -> Synchronizing -> Rebuilding -> Ready (+ Recovering)
export type SyncState =
  | "disconnected"
  | "connecting"
  | "handshaking"
  | "synchronizing"
  | "rebuilding"
  | "ready"
  | "recovering";

export interface SyncFence {
  profileId: string;
  contextId: string;
  epoch: number;
  revision: number;
  sequence: number;
}

export class ProjectionSyncController {
  private state: SyncState = "disconnected";
  private fence: SyncFence | null = null;

  getState(): SyncState { return this.state; }
  getFence(): SyncFence | null { return this.fence; }

  // Reject tab data before a successful hello_ack.
  canAcceptTabData(): boolean { return this.state === "ready" || this.state === "synchronizing"; }

  transition(next: SyncState): void { this.state = next; }

  setFence(fence: SyncFence): void { this.fence = fence; }

  reset(): void { this.state = "disconnected"; this.fence = null; }
}
