import type { Room } from './types';

export type ConsensusState = { attempt: string; revealed: boolean; value: string | null };

export function consensusState(room: Room): ConsensusState {
  const round = room.rounds.find((item) => item.id === room.active_round_id);
  const revealed = round?.status === 'revealed' || round?.status === 'completed';
  const voters = room.members.filter((member) => !member.observer && member.online);
  const value = voters[0]?.vote;
  const unanimous =
    !room.closed &&
    revealed &&
    voters.length > 1 &&
    !!value &&
    value !== '?' &&
    value !== '☕' &&
    room.cards.includes(value) &&
    voters.every((member) => member.has_voted && member.vote === value);

  return {
    attempt: `${room.id}:${round?.id || ''}:${round?.started_at || ''}`,
    revealed,
    value: unanimous ? value : null,
  };
}

export function shouldCelebrate(previous: ConsensusState, current: ConsensusState): boolean {
  return (
    previous.attempt === current.attempt &&
    !previous.revealed &&
    current.revealed &&
    current.value !== null
  );
}
