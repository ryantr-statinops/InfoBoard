// Projection synchronization state machine for InfoBoard (IP-09).
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

export interface HelloAck {
  protocolVersion: string;
  profileId: string;
  persistenceDegraded: boolean;
}

export class ProjectionSyncController {
  private state: SyncState = "disconnected";
  private fence: SyncFence | null = null;
  private helloAck: HelloAck | null = null;

  getState(): SyncState { return this.state; }
  getFence(): SyncFence | null { return this.fence; }
  getHelloAck(): HelloAck | null { return this.helloAck; }

  canAcceptTabData(): boolean {
    return this.state === "ready" || this.state === "synchronizing";
  }

  // T02: hello-to-ready gating entry.
  async startHello(profileId: string, sendHello: (p: string) => Promise<HelloAck>): Promise<void> {
    this.transition("connecting");
    this.transition("handshaking");
    const ack = await sendHello(profileId);
    if (ack.profileId !== profileId) throw new Error("PROFILE_MISMATCH");
    this.helloAck = ack;
    this.transition("synchronizing");
  }

  transition(next: SyncState): void { this.state = next; }

  setFence(fence: SyncFence): void { this.fence = fence; }

  markReady(fence: SyncFence): void {
    this.fence = fence;
    this.transition("ready");
  }

  reset(): void { this.state = "disconnected"; this.fence = null; this.helloAck = null; }
}
