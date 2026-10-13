import { useEffect, useRef, useState } from 'react';
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  Globe2,
  LayoutGrid,
  ListTodo,
  Loader2,
  PanelRightClose,
  Plus,
  RefreshCw,
  Search,
  Users,
} from 'lucide-react';
import { api } from './api';
import { roomTheme } from './themes';
import type { PublicRoom, Room, User } from './types';

const fa = (n: number) => n.toLocaleString('fa-IR');
const pageSize = 6;

export function RoomsDock({
  user,
  room,
  demo,
  collapsed,
  onCollapse,
  onCreate,
  onOpen,
}: {
  user: User | null;
  room: Room | null;
  demo: boolean;
  collapsed: boolean;
  onCollapse: (value: boolean) => void;
  onCreate: () => void;
  onOpen: (id: string) => void;
}) {
  const [rooms, setRooms] = useState<Room[]>([]);
  const [publicRooms, setPublicRooms] = useState<PublicRoom[]>([]);
  const [tab, setTab] = useState<'mine' | 'public'>(user || demo ? 'mine' : 'public');
  const [filter, setFilter] = useState('active');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0);
  const [refresh, setRefresh] = useState(0);
  const [loadingMine, setLoadingMine] = useState(!!user);
  const [loadingPublic, setLoadingPublic] = useState(false);
  const [errorMine, setErrorMine] = useState('');
  const [errorPublic, setErrorPublic] = useState('');
  const toggleRef = useRef<HTMLButtonElement>(null);
  const previousCollapsed = useRef(collapsed);

  useEffect(() => {
    if (previousCollapsed.current !== collapsed) toggleRef.current?.focus();
    previousCollapsed.current = collapsed;
  }, [collapsed]);

  useEffect(() => {
    if (!user) return;
    let alive = true;
    let pending = false;
    setLoadingMine(true);
    const load = async () => {
      if (pending) return;
      pending = true;
      try {
        const next = await api<Room[]>('/rooms');
        if (alive) {
          setRooms(next.sort((a, b) => b.created_at.localeCompare(a.created_at)));
          setErrorMine('');
        }
      } catch (e) {
        if (alive) setErrorMine((e as Error).message);
      } finally {
        pending = false;
        if (alive) setLoadingMine(false);
      }
    };
    void load();
    const timer = window.setInterval(() => {
      if (!document.hidden) void load();
    }, 30000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [user?.id, room?.id, room?.closed, refresh]);

  useEffect(() => {
    if (tab !== 'public' || demo) return;
    let alive = true;
    let pending = false;
    setLoadingPublic(true);
    setPublicRooms([]);
    const load = async () => {
      if (pending) return;
      pending = true;
      try {
        const next = await api<PublicRoom[]>(
          `/rooms/public?limit=${pageSize + 1}&offset=${page * pageSize}`,
        );
        if (alive) {
          setPublicRooms(next);
          setErrorPublic('');
        }
      } catch (e) {
        if (alive) setErrorPublic((e as Error).message);
      } finally {
        pending = false;
        if (alive) setLoadingPublic(false);
      }
    };
    void load();
    const timer = window.setInterval(() => {
      if (!document.hidden) void load();
    }, 30000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [tab, page, refresh, demo]);

  const mine = room ? [room, ...rooms.filter((item) => item.id !== room.id)] : rooms;
  const shown = mine.filter(
    (item) =>
      item.name.includes(search) &&
      (filter === 'all' || (filter === 'past' ? item.closed : !item.closed)),
  );
  const loading = tab === 'mine' ? loadingMine : loadingPublic;
  const error = tab === 'mine' ? errorMine : errorPublic;

  if (collapsed) {
    return (
      <button
        ref={toggleRef}
        className="workspace-floating-button rooms-floating-button"
        aria-label="باز کردن اتاق‌ها"
        aria-expanded={false}
        aria-controls="rooms-dock"
        title="اتاق‌ها"
        onClick={() => onCollapse(false)}
      >
        <LayoutGrid size={23} />
        {room && <i className="live-dot" />}
      </button>
    );
  }

  return (
    <>
      <button
        className="rooms-dock-backdrop"
        aria-label="بستن کارت اتاق‌ها"
        onClick={() => onCollapse(true)}
      />
      <aside
        id="rooms-dock"
        className="rooms-dock"
        aria-labelledby="rooms-dock-title"
        onKeyDown={(e) => {
          if (e.key === 'Escape') onCollapse(true);
        }}
      >
        <div className="rooms-dock-heading">
          <div>
            <span className="eyebrow">فضای تیم</span>
            <h2 id="rooms-dock-title">
              <LayoutGrid size={19} /> اتاق‌ها
            </h2>
          </div>
          <button
            ref={toggleRef}
            className="icon-button"
            aria-label="جمع کردن اتاق‌ها"
            aria-expanded={true}
            aria-controls="rooms-dock"
            title="جمع کردن اتاق‌ها"
            onClick={() => onCollapse(true)}
          >
            <PanelRightClose size={19} />
          </button>
        </div>
        <button className="primary full" onClick={onCreate}>
          <Plus size={17} /> اتاق جدید
        </button>
        {!demo && (
          <div className="rooms-dock-tabs" aria-label="فهرست اتاق‌ها">
            {user && (
              <button
                className={tab === 'mine' ? 'selected' : ''}
                aria-pressed={tab === 'mine'}
                onClick={() => setTab('mine')}
              >
                جلسه‌های من
              </button>
            )}
            <button
              className={tab === 'public' ? 'selected' : ''}
              aria-pressed={tab === 'public'}
              onClick={() => setTab('public')}
            >
              <Globe2 size={15} /> عمومی
            </button>
          </div>
        )}
        {tab === 'mine' && (
          <>
            <div className="search rooms-dock-search">
              <Search size={16} />
              <input
                aria-label="جستجوی جلسه"
                placeholder="جستجوی جلسه…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <div className="rooms-dock-filters" aria-label="وضعیت جلسه‌ها">
              {[
                ['active', 'فعال'],
                ['past', 'تمام‌شده'],
                ['all', 'همه'],
              ].map(([value, label]) => (
                <button
                  key={value}
                  className={filter === value ? 'selected' : ''}
                  aria-pressed={filter === value}
                  onClick={() => setFilter(value)}
                >
                  {label}
                </button>
              ))}
            </div>
          </>
        )}
        <div className="rooms-dock-list" tabIndex={0} aria-label="اتاق‌های در دسترس">
          {error ? (
            <div className="rooms-dock-empty" role="alert">
              <p className="form-error">{error}</p>
              <button className="secondary small-button" onClick={() => setRefresh((v) => v + 1)}>
                تلاش دوباره
              </button>
            </div>
          ) : loading ? (
            <div className="rooms-dock-empty" role="status">
              <Loader2 className="spin" size={22} />
              <p>در حال دریافت اتاق‌ها…</p>
            </div>
          ) : tab === 'mine' ? (
            shown.length ? (
              shown.map((item) => (
                <button
                  key={item.id}
                  className={`rooms-dock-room ${item.id === room?.id ? 'current' : ''}`}
                  data-palette={roomTheme(item.theme)}
                  aria-label={`ورود به ${item.name}`}
                  aria-current={item.id === room?.id ? 'page' : undefined}
                  onClick={() => onOpen(item.id)}
                >
                  <span className="rooms-dock-room-top">
                    <span className={`badge ${item.closed ? 'gray' : 'green'}`}>
                      {item.closed
                        ? 'پایان‌یافته'
                        : item.id === room?.id
                          ? 'در این اتاق هستید'
                          : 'فعال'}
                    </span>
                    <ArrowLeft size={15} />
                  </span>
                  <strong>{item.name}</strong>
                  <span className="rooms-dock-room-meta">
                    <span>
                      <Users size={13} />
                      {fa(item.members.filter((member) => member.online).length)} نفر
                    </span>
                    <span>
                      <ListTodo size={13} />
                      {fa(item.rounds.length)} تسک
                    </span>
                  </span>
                </button>
              ))
            ) : (
              <div className="rooms-dock-empty">
                <LayoutGrid size={25} />
                <p>{search ? 'جلسه‌ای پیدا نشد' : 'هنوز جلسه‌ای در این بخش ندارید'}</p>
                <small>اتاق بسازید یا به یک اتاق عمومی بپیوندید.</small>
              </div>
            )
          ) : publicRooms.length ? (
            publicRooms.slice(0, pageSize).map((item) => {
              const member = mine.some((joined) => joined.id === item.id);
              const full = item.member_count >= item.capacity;
              return (
                <button
                  key={item.id}
                  className={`rooms-dock-room ${item.id === room?.id ? 'current' : ''}`}
                  data-palette={roomTheme(item.theme)}
                  aria-label={`${member ? 'ورود به' : 'پیوستن به'} ${item.name}`}
                  aria-current={item.id === room?.id ? 'page' : undefined}
                  disabled={full && !member}
                  onClick={() => onOpen(item.id)}
                >
                  <span className="rooms-dock-room-top">
                    <span className={`badge ${full ? 'gray' : 'green'}`}>
                      <Globe2 size={12} />
                      {full ? 'ظرفیت تکمیل' : 'عمومی'}
                    </span>
                    <ArrowLeft size={15} />
                  </span>
                  <strong>{item.name}</strong>
                  <small>گرداننده: {item.host_name}</small>
                  <span className="rooms-dock-room-meta">
                    <span>
                      <Users size={13} />
                      {fa(item.member_count)} از {fa(item.capacity)} نفر
                    </span>
                    <span>{item.deck === 'fibonacci' ? 'فیبوناچی' : 'تی‌شرت'}</span>
                  </span>
                </button>
              );
            })
          ) : (
            <div className="rooms-dock-empty">
              <Globe2 size={25} />
              <p>{page ? 'اتاق دیگری نیست' : 'هنوز اتاق عمومی باز نشده'}</p>
            </div>
          )}
        </div>
        {tab === 'public' && (page > 0 || publicRooms.length > pageSize) && (
          <div className="rooms-dock-pagination" aria-label="صفحه‌بندی اتاق‌های عمومی">
            <button
              className="icon-button"
              aria-label="صفحه قبلی اتاق‌ها"
              disabled={!page || loading}
              onClick={() => setPage((v) => v - 1)}
            >
              <ChevronRight size={17} />
            </button>
            <span>صفحهٔ {fa(page + 1)}</span>
            <button
              className="icon-button"
              aria-label="صفحه بعدی اتاق‌ها"
              disabled={publicRooms.length <= pageSize || loading}
              onClick={() => setPage((v) => v + 1)}
            >
              <ChevronLeft size={17} />
            </button>
          </div>
        )}
        <button
          className="text-button rooms-dock-refresh"
          disabled={loading}
          onClick={() => setRefresh((v) => v + 1)}
        >
          <RefreshCw size={14} className={loading ? 'spin' : ''} /> تازه‌سازی اتاق‌ها
        </button>
      </aside>
    </>
  );
}
