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

// Allowed transitions keep the sync lifecycle monotonic and observable.
const TRANSITIONS: Record<SyncState, readonly SyncState[]> = {
  disconnected: ["connecting"],
  connecting: ["handshaking", "disconnected", "recovering"],
  handshaking: ["synchronizing", "disconnected", "recovering"],
  synchronizing: ["rebuilding", "ready", "disconnected", "recovering"],
  rebuilding: ["ready", "disconnected", "recovering"],
  ready: ["rebuilding", "disconnected", "recovering"],
  recovering: ["connecting", "disconnected", "rebuilding", "ready"],
};

export class ProjectionSyncController {
  private state: SyncState = "disconnected";
  private fence: SyncFence | null = null;
  private helloAck: HelloAck | null = null;
  private lastError: string | null = null;

  getState(): SyncState { return this.state; }
  getFence(): SyncFence | null { return this.fence; }
  getHelloAck(): HelloAck | null { return this.helloAck; }
  getLastError(): string | null { return this.lastError; }

  getPersistenceDegraded(): boolean {
    return this.helloAck?.persistenceDegraded ?? false;
  }

  canAcceptTabData(): boolean {
    return this.state === "ready" || this.state === "synchronizing";
  }

  transition(next: SyncState): void {
    const allowed = TRANSITIONS[this.state];
    if (!allowed.includes(next)) {
      this.lastError = `INVALID_TRANSITION:${this.state}->${next}`;
      throw new Error(this.lastError);
    }
    this.state = next;
  }

  setFence(fence: SyncFence): void { this.fence = fence; }

  // T02: hello-to-ready gating entry.
  async startHello(profileId: string, sendHello: (p: string) => Promise<HelloAck>): Promise<void> {
    this.transition("connecting");
    this.transition("handshaking");
    const ack = await sendHello(profileId);
    if (ack.profileId !== profileId) {
      this.lastError = "PROFILE_MISMATCH";
      throw new Error(this.lastError);
    }
    this.helloAck = ack;
    this.transition("synchronizing");
  }

  markReady(fence: SyncFence): void {
    this.fence = fence;
    this.transition("ready");
  }

  fail(error: string): void {
    this.lastError = error;
    this.transition("recovering");
  }

  reset(): void {
    this.state = "disconnected";
    this.fence = null;
    this.helloAck = null;
    this.lastError = null;
  }
}
