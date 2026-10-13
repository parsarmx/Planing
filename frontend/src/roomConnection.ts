import type { Room } from './types';

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'expired';

// Presence is confirmed by a room snapshot, rather than merely opening the HTTP stream.
export function openRoomConnection(
  source: EventSource,
  userID: string,
  callbacks: {
    room: (room: Room) => void;
    disconnected: (reason: string) => void;
    expired: () => void;
  },
  timeout = 45_000,
) {
  let stopped = false;
  let timer: ReturnType<typeof setTimeout>;
  const stop = () => {
    stopped = true;
    clearTimeout(timer);
    source.close();
  };
  const disconnect = (reason: string, terminal = true) => {
    if (stopped) return;
    clearTimeout(timer);
    if (terminal) stop();
    callbacks.disconnected(reason);
  };
  const arm = () => {
    clearTimeout(timer);
    timer = setTimeout(() => disconnect('پاسخی از اتاق دریافت نشد.'), timeout);
  };
  source.addEventListener('room', (event) => {
    if (stopped) return;
    let room: Room;
    try {
      room = JSON.parse((event as MessageEvent).data);
    } catch {
      disconnect('اطلاعات اتصال معتبر نیست.');
      return;
    }
    if (!room || !Array.isArray(room.members)) {
      disconnect('اطلاعات اتصال معتبر نیست.');
      return;
    }
    if (!room.closed && !room.members.some((member) => member.id === userID && member.online)) {
      disconnect('حضور شما در اتاق تأیید نشد.');
      return;
    }
    arm();
    callbacks.room(room);
    if (room.closed) stop();
  });
  source.addEventListener('idle', () => disconnect('اتصال جلسه پس از بی‌فعالیتی بسته شد.'));
  source.addEventListener('unavailable', () => disconnect('اتاق در حال حاضر در دسترس نیست.'));
  source.addEventListener('expired', () => {
    if (stopped) return;
    stop();
    callbacks.expired();
  });
  source.onerror = () => {
    // EventSource retries automatically; the room stays hidden until a fresh snapshot arrives.
    disconnect('ارتباط با اتاق قطع شده است.', false);
    if (!stopped) arm();
  };
  arm();
  return stop;
}
