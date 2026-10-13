import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { runInNewContext } from 'node:vm';
import test from 'node:test';
import ts from 'typescript';

// Run the same pure TypeScript rules used by the UI, without a browser or extra runtime.
const source = await readFile(new URL('../src/consensus.ts', import.meta.url), 'utf8');
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
});
const compiled = { exports: {} };
runInNewContext(outputText, compiled, { filename: 'consensus.js' });
const { consensusState, shouldCelebrate } = compiled.exports;

function room(values, status = 'revealed', observers = []) {
  return {
    id: 'room',
    closed: false,
    active_round_id: 'round',
    cards: ['0', '5', '8', 'XS', 'M', '?', '☕'],
    rounds: [{ id: 'round', status, started_at: 'attempt-1' }],
    members: [
      ...values.map((vote, i) => ({
        id: String(i),
        observer: false,
        online: true,
        has_voted: vote !== undefined,
        vote,
      })),
      ...observers,
    ],
  };
}

test('all voters must agree on a revealed estimate', () => {
  assert.equal(consensusState(room(['5', '5', '5'])).value, '5');
});
test('hidden votes never trigger a celebration', () => {
  assert.equal(consensusState(room(['5', '5'], 'voting')).value, null);
});
test('matching votes from only part of the team are insufficient', () => {
  assert.equal(consensusState(room(['5', '5', undefined])).value, null);
});
test('different estimates do not celebrate', () => {
  assert.equal(consensusState(room(['5', '8'])).value, null);
});
test('observers do not need to cast a vote', () => {
  assert.equal(
    consensusState(room(['5', '5'], 'revealed', [{ observer: true, has_voted: false }])).value,
    '5',
  );
});
test('zero is a valid shared estimate', () => {
  assert.equal(consensusState(room(['0', '0'])).value, '0');
});
test('T-shirt estimates can also agree', () => {
  assert.equal(consensusState(room(['M', 'M'])).value, 'M');
});
test('unknown and coffee cards are not shared estimates', () => {
  for (const value of ['?', '☕']) assert.equal(consensusState(room([value, value])).value, null);
});
test('empty rooms and a single voter do not celebrate team consensus', () => {
  for (const values of [[], ['5']]) assert.equal(consensusState(room(values)).value, null);
});
test('estimates outside the deck are rejected', () => {
  assert.equal(consensusState(room(['invalid', 'invalid'])).value, null);
});
test('closed sessions do not celebrate', () => {
  assert.equal(consensusState({ ...room(['5', '5']), closed: true }).value, null);
});
test('the transition from voting to reveal starts the celebration', () => {
  assert.equal(
    shouldCelebrate(consensusState(room(['5', '5'], 'voting')), consensusState(room(['5', '5']))),
    true,
  );
});
test('SSE updates and registering the estimate do not replay it', () => {
  const current = consensusState(room(['5', '5']));
  assert.equal(shouldCelebrate(current, { ...current }), false);
  assert.equal(shouldCelebrate(current, consensusState(room(['5', '5'], 'completed'))), false);
});
test('opening historical results does not replay it', () => {
  const current = consensusState(room(['5', '5']));
  assert.equal(shouldCelebrate(current, current), false);
});
test('a different round cannot replay previous consensus', () => {
  const previous = consensusState(room(['5', '5'], 'voting'));
  const current = { ...consensusState(room(['5', '5'])), attempt: 'other-round' };
  assert.equal(shouldCelebrate(previous, current), false);
});
test('a reset permits a new celebration after another reveal', () => {
  const fixture = room(['5', '5'], 'voting');
  fixture.rounds[0].started_at = 'attempt-2';
  const previous = consensusState(fixture);
  fixture.rounds[0].status = 'revealed';
  assert.equal(shouldCelebrate(previous, consensusState(fixture)), true);
});

test('disconnected participants are excluded from the current electorate', () => {
  const fixture = room(['5', '5', '8']);
  fixture.members[2].online = false;
  assert.equal(consensusState(fixture).value, '5');
  fixture.members[2].online = true;
  assert.equal(consensusState(fixture).value, null);
});
