import test from "node:test";
import assert from "node:assert/strict";
import { ProjectionSyncController } from "../../../extension/dist/src/runtime/projection-sync.js";

const ack = (profileId) => ({ protocolVersion: "1", profileId, persistenceDegraded: false });

test("snapshot is refused before hello_ack", async () => {
  const sync = new ProjectionSyncController();
  await assert.rejects(() => sync.sendSnapshot(async () => "sent"), /SNAPSHOT_BEFORE_HELLO_ACK/);
});

test("profile mismatch is refused", async () => {
  const sync = new ProjectionSyncController();
  await assert.rejects(() => sync.startHello("p1", async () => ack("p2")), /PROFILE_MISMATCH/);
});

test("valid hello enables snapshot and reaches synchronizing", async () => {
  const sync = new ProjectionSyncController();
  await sync.startHello("p1", async () => ack("p1"));
  assert.equal(sync.getState(), "synchronizing");
  assert.equal(await sync.sendSnapshot(async () => "sent"), "sent");
});

test("invalid transition is rejected", () => {
  const sync = new ProjectionSyncController();
  assert.throws(() => sync.transition("ready"), /INVALID_TRANSITION/);
});
