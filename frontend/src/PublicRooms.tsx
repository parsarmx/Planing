import { useEffect, useState } from 'react';
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  Globe2,
  Loader2,
  RefreshCw,
  Users,
} from 'lucide-react';
import { api } from './api';
import { roomTheme } from './themes';
import type { PublicRoom } from './types';

const pageSize = 12;
const fa = (n: number) => n.toLocaleString('fa-IR');

export function PublicRooms({
  onJoin,
  joined = [],
}: {
  onJoin: (id: string) => void;
  joined?: string[];
}) {
  const [rooms, setRooms] = useState<PublicRoom[]>([]);
  const [page, setPage] = useState(0);
  const [refresh, setRefresh] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  useEffect(() => {
    let alive = true;
    let pending = false;
    setLoading(true);
    setRooms([]);
    const load = async () => {
      if (pending) return;
      pending = true;
      try {
        const next = await api<PublicRoom[]>(
          `/rooms/public?limit=${pageSize + 1}&offset=${page * pageSize}`,
        );
        if (alive) {
          setRooms(next);
          setError('');
        }
      } catch (e) {
        if (alive) setError((e as Error).message);
      } finally {
        pending = false;
        if (alive) setLoading(false);
      }
    };
    void load();
    const timer = window.setInterval(() => {
      if (!document.hidden) void load();
    }, 30000);
    const visible = () => {
      if (!document.hidden) void load();
    };
    document.addEventListener('visibilitychange', visible);
    return () => {
      alive = false;
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', visible);
    };
  }, [page, refresh]);
  return (
    <section className="public-rooms" aria-labelledby="public-rooms-title">
      <div className="public-rooms-heading">
        <div>
          <span className="eyebrow">
            <Globe2 size={16} /> یک صندلی برای همه
          </span>
          <h2 id="public-rooms-title">اتاق‌های عمومی</h2>
          <p className="muted">یک میز انتخاب کن و به گفت‌وگو بپیوند؛ بدون نیاز به لینک دعوت.</p>
        </div>
        <button
          type="button"
          className="secondary small-button"
          disabled={loading}
          onClick={() => setRefresh((v) => v + 1)}
          aria-label="تازه‌سازی اتاق‌های عمومی"
        >
          <RefreshCw size={16} className={loading ? 'spin' : ''} /> تازه‌سازی
        </button>
      </div>
      {error ? (
        <div className="public-rooms-status" role="alert">
          <p className="form-error">{error}</p>
          <button className="secondary" onClick={() => setRefresh((v) => v + 1)}>
            تلاش دوباره
          </button>
        </div>
      ) : loading ? (
        <div className="public-rooms-status" role="status">
          <Loader2 className="spin" size={24} /> در حال پیدا کردن میزها…
        </div>
      ) : rooms.length ? (
        <div className="room-grid public-room-grid">
          {rooms.slice(0, pageSize).map((room) => {
            const full = room.member_count >= room.capacity;
            const member = joined.includes(room.id);
            return (
              <button
                type="button"
                className="room-tile public-room-tile"
                data-palette={roomTheme(room.theme)}
                key={room.id}
                disabled={full && !member}
                onClick={() => onJoin(room.id)}
                aria-label={`${member ? 'ورود به' : 'پیوستن به'} ${room.name}`}
              >
                <div className="room-tile-top">
                  <span className="tile-icon">
                    <Globe2 size={22} />
                  </span>
                  <span className={`badge ${full ? 'gray' : 'green'}`}>
                    {full ? 'ظرفیت تکمیل' : 'عمومی'}
                  </span>
                </div>
                <h3>{room.name}</h3>
                <p className="public-host">گرداننده: {room.host_name}</p>
                <div className="public-room-meta">
                  <span>
                    <Users size={16} /> {fa(room.member_count)} از {fa(room.capacity)} نفر
                  </span>
                  <span>{room.deck === 'fibonacci' ? 'فیبوناچی' : 'اندازه تی‌شرت'}</span>
                </div>
                <div className="public-join">
                  <span>
                    {full && !member ? 'اتاق پر است' : member ? 'بازگشت به اتاق' : 'پیوستن به اتاق'}
                  </span>
                  <ArrowLeft size={18} />
                </div>
              </button>
            );
          })}
        </div>
      ) : (
        <div className="public-rooms-status">
          <Globe2 size={28} />
          <h3>{page ? 'اتاق دیگری نیست' : 'هنوز میز عمومی باز نشده'}</h3>
          <p className="muted">با ساخت یک اتاق عمومی، میزت را به این جمع اضافه کن.</p>
        </div>
      )}
      {(page > 0 || rooms.length > pageSize) && (
        <div className="public-pagination" aria-label="صفحه‌بندی اتاق‌های عمومی">
          <button
            className="secondary small-button"
            disabled={page === 0 || loading}
            onClick={() => setPage((v) => v - 1)}
          >
            <ChevronRight size={16} /> قبلی
          </button>
          <span>صفحهٔ {fa(page + 1)}</span>
          <button
            className="secondary small-button"
            disabled={rooms.length <= pageSize || loading}
            onClick={() => setPage((v) => v + 1)}
          >
            بعدی <ChevronLeft size={16} />
          </button>
        </div>
      )}
    </section>
  );
}
