import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { runInNewContext } from 'node:vm';
import test from 'node:test';
import ts from 'typescript';
const source = await readFile(new URL('../src/roomConnection.ts', import.meta.url), 'utf8');
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
});
function fixture() {
  const timers = new Map();
  let id = 0;
  const compiled = {
    exports: {},
    setTimeout: (fn) => {
      timers.set(++id, fn);
      return id;
    },
    clearTimeout: (id) => timers.delete(id),
  };
  runInNewContext(outputText, compiled);
  const source = {
    listeners: {},
    closed: false,
    addEventListener(name, fn) {
      this.listeners[name] = fn;
    },
    close() {
      this.closed = true;
    },
  };
  const seen = { room: [], disconnected: [], expired: 0 };
  const stop = compiled.exports.openRoomConnection(source, 'me', {
    room: (r) => seen.room.push(r),
    disconnected: (r) => seen.disconnected.push(r),
    expired: () => seen.expired++,
  });
  const snapshot = (online = true, closed = false) =>
    source.listeners.room({ data: JSON.stringify({ closed, members: [{ id: 'me', online }] }) });
  return { source, seen, timers, stop, snapshot };
}
test('opening the stream does not enter the room until presence is confirmed', () => {
  const f = fixture();
  assert.equal(f.seen.room.length, 0);
  f.snapshot();
  assert.equal(f.seen.room.length, 1);
  f.stop();
});
test('offline snapshots cannot grant entry', () => {
  const f = fixture();
  f.snapshot(false);
  assert.equal(f.seen.room.length, 0);
  assert.equal(f.seen.disconnected.length, 1);
  assert.equal(f.source.closed, true);
});
test('network errors hide the room and a fresh automatic retry restores it', () => {
  const f = fixture();
  f.snapshot();
  f.source.onerror();
  assert.equal(f.seen.disconnected.length, 1);
  assert.equal(f.source.closed, false);
  f.snapshot();
  assert.equal(f.seen.room.length, 2);
  f.stop();
});
test('a stalled heartbeat closes the connection', () => {
  const f = fixture();
  f.snapshot();
  [...f.timers.values()][0]();
  assert.equal(f.source.closed, true);
  assert.equal(f.seen.disconnected.length, 1);
  f.snapshot();
  assert.equal(f.seen.room.length, 1);
});
test('expired sessions stop reconnecting and request authentication', () => {
  const f = fixture();
  f.source.listeners.expired();
  assert.equal(f.seen.expired, 1);
  assert.equal(f.source.closed, true);
  f.snapshot();
  assert.equal(f.seen.room.length, 0);
});
test('idle and unavailable terminal events require a fresh connection', () => {
  for (const event of ['idle', 'unavailable']) {
    const f = fixture();
    f.source.listeners[event]();
    assert.equal(f.source.closed, true);
    assert.equal(f.seen.disconnected.length, 1);
  }
});
test('closed rooms retain their final snapshot and stop streaming', () => {
  const f = fixture();
  f.snapshot(false, true);
  assert.equal(f.seen.room.length, 1);
  assert.equal(f.source.closed, true);
  assert.equal(f.seen.disconnected.length, 0);
});
test('cleanup releases the stream and ignores subsequent callbacks', () => {
  const f = fixture();
  f.stop();
  assert.equal(f.timers.size, 0);
  f.snapshot();
  f.source.onerror();
  assert.equal(f.seen.room.length, 0);
  assert.equal(f.seen.disconnected.length, 0);
});

test('invalid snapshots disconnect without leaving an active voting screen', () => {
  for (const data of ['not json', '{}', 'null']) {
    const f = fixture();
    f.source.listeners.room({ data });
    assert.equal(f.source.closed, true);
    assert.equal(f.seen.room.length, 0);
    assert.equal(f.seen.disconnected.length, 1);
  }
});
