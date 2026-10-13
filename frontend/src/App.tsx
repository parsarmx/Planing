import { useState, useEffect, useRef, type ReactNode, type FormEvent } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Plus,
  Users,
  Check,
  ChevronLeft,
  Clock3,
  Copy,
  Crown,
  Eye,
  EyeOff,
  LogOut,
  Settings2,
  X,
  Search,
  Sparkles,
  RotateCcw,
  Play,
  CheckCircle2,
  ListTodo,
  Download,
  CircleHelp,
  Link2,
  Smartphone,
  ShieldCheck,
  Coffee,
  LayoutGrid,
  ChartNoAxesColumn,
  ChevronUp,
  Minimize2,
  Loader2,
  CircleDashed,
  Flag,
  Heart,
  WifiOff,
} from 'lucide-react';
import type { User, Room, Round, Command } from './types';
import { PublicRooms } from './PublicRooms';
import { RoomsDock } from './RoomsDock';
import { ConsensusCelebration } from './ConsensusCelebration';
import { consensusState, shouldCelebrate } from './consensus';
import { openRoomConnection, type ConnectionStatus } from './roomConnection';
import { VisibilityPicker } from './VisibilityPicker';
import { api, APIError } from './api';
import { demoRoom, demoCommand, demoUser } from './demo';
import { Palette } from 'lucide-react';
import { ThemePicker, roomTheme, type RoomTheme } from './themes';
import { Avatar, AvatarBuilder, avatarFor } from './avatars';
const fa = (n: number) => n.toLocaleString('fa-IR');
const statusText: Record<string, string> = {
  pending: 'در انتظار',
  voting: 'در حال رأی‌گیری',
  revealed: 'کارت‌ها رو شده',
  completed: 'تخمین زده شد',
};
function Brand() {
  return (
    <span className="brand">
      <span className="brand-icon">
        <span />
        <span />
        <Sparkles size={15} />
      </span>
      <span>
        هم‌تخمین<span className="brand-sub">planning poker</span>
      </span>
    </span>
  );
}
function Modal({
  title,
  children,
  onClose,
  wide = false,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [viewport, setViewport] = useState(() => {
    const view = window.visualViewport;
    return view ? { height: view.height, top: view.offsetTop } : null;
  });
  useEffect(() => {
    const view = window.visualViewport;
    if (!view) return;
    const update = () => setViewport({ height: view.height, top: view.offsetTop });
    view.addEventListener('resize', update);
    view.addEventListener('scroll', update);
    update();
    return () => {
      view.removeEventListener('resize', update);
      view.removeEventListener('scroll', update);
    };
  }, []);
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const node = ref.current;
    const input = node?.querySelector<HTMLElement>('input:not([readonly]),textarea');
    if (input) input.focus();
    else node?.focus();
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeRef.current();
      if (e.key === 'Tab' && node) {
        const els = Array.from(
          node.querySelectorAll<HTMLElement>(
            'button:not(:disabled),input:not(:disabled),select:not(:disabled),textarea:not(:disabled),a[href]',
          ),
        );
        const first = els[0],
          last = els[els.length - 1];
        if (e.shiftKey && (document.activeElement === first || document.activeElement === node)) {
          e.preventDefault();
          last?.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first?.focus();
        }
      }
    };
    document.addEventListener('keydown', handler);
    const old = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', handler);
      document.body.style.overflow = old;
      previous?.focus();
    };
  }, []);
  return (
    <div
      className="modal-backdrop"
      style={viewport ? { top: viewport.top, height: viewport.height, bottom: 'auto' } : undefined}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className={`modal ${wide ? 'wide' : ''}`}
        ref={ref}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-label={title}
      >
        <div className="modal-title">
          <h2>{title}</h2>
          <button className="icon-button" aria-label="بستن" onClick={onClose}>
            <X size={20} />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
function Button({
  children,
  busy = false,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { busy?: boolean }) {
  return (
    <button {...props} disabled={props.disabled || busy}>
      {busy ? <Loader2 className="spin" size={18} /> : null}
      {children}
    </button>
  );
}
function AuthModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: (u: User) => void }) {
  const [phone, setPhone] = useState(''),
    [code, setCode] = useState(''),
    [dev, setDev] = useState(''),
    [step, setStep] = useState(0),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(''),
    [wait, setWait] = useState(0);
  useEffect(() => {
    if (wait <= 0) return;
    const t = setTimeout(() => setWait((x) => x - 1), 1000);
    return () => clearTimeout(t);
  }, [wait]);
  async function send(e?: FormEvent) {
    e?.preventDefault();
    setBusy(true);
    setError('');
    try {
      const ch = await api<{ phone: string; dev_code?: string; retry_after: number }>(
        '/auth/otp',
        'POST',
        { phone },
      );
      setPhone(ch.phone);
      setDev(ch.dev_code || '');
      setWait(ch.retry_after);
      setCode('');
      setStep(1);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function verify(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      onSuccess(await api<User>('/auth/verify', 'POST', { phone, code }));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title={step ? 'کد تأیید را وارد کنید' : 'به هم‌تخمین خوش آمدید'} onClose={onClose}>
      <div className="auth-icon">
        <Smartphone size={30} />
      </div>
      <p className="muted">
        {step
          ? `کد ۶ رقمی برای ${phone} آماده شده است.`
          : 'با شماره موبایل وارد شوید و تخمین بعدی را با تیم شروع کنید.'}
      </p>
      <form onSubmit={step ? verify : send}>
        {step ? (
          <>
            <label>
              کد یک‌بارمصرف
              <input
                autoFocus
                aria-label="کد یک‌بارمصرف"
                dir="ltr"
                className="otp-input"
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                value={code}
                onChange={(e) =>
                  setCode(
                    e.target.value
                      .replace(/[۰-۹]/g, (c) => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(c)))
                      .replace(/[^0-9]/g, ''),
                  )
                }
                placeholder="• • • • • •"
                required
              />
            </label>
            {dev && (
              <div className="dev-note">
                <ShieldCheck size={17} />
                <div>
                  حالت توسعه؛ پیامک ارسال نمی‌شود.
                  <br />
                  <button type="button" onClick={() => setCode(dev)}>
                    کد آزمایشی: <b dir="ltr">{dev}</b> · استفاده از کد
                  </button>
                </div>
              </div>
            )}
          </>
        ) : (
          <label>
            شماره موبایل
            <input
              autoFocus
              aria-label="شماره موبایل"
              dir="ltr"
              inputMode="tel"
              autoComplete="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="0912 345 6789"
              maxLength={20}
              required
            />
          </label>
        )}
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <Button className="primary full" busy={busy} type="submit">
          {step ? 'تأیید و ورود' : 'دریافت کد ورود'}
          <ArrowLeft size={17} />
        </Button>
      </form>
      {step > 0 && (
        <div className="auth-links">
          <button
            className="text-button"
            onClick={() => {
              setStep(0);
              setError('');
            }}
          >
            تغییر شماره
          </button>
          <Button className="text-button" busy={busy} disabled={wait > 0} onClick={() => send()}>
            {wait ? `ارسال دوباره تا ${fa(wait)} ثانیه` : 'ارسال دوباره کد'}
          </Button>
        </div>
      )}
      <div className="secure-note">
        <ShieldCheck size={15} />
        ورود امن، بدون نیاز به رمز عبور
      </div>
    </Modal>
  );
}
function ProfileModal({
  user,
  onClose,
  onSave,
}: {
  user: User;
  onClose: () => void;
  onSave: (u: User) => void;
}) {
  const [name, setName] = useState(user.name),
    [role, setRole] = useState(user.role),
    [color, setColor] = useState(user.avatar || 'violet'),
    [appearance, setAppearance] = useState(user.avatar_config || avatarFor(user.name || user.id)),
    [editingAvatar, setEditingAvatar] = useState(false),
    [error, setError] = useState(''),
    [busy, setBusy] = useState(false);
  async function save(e: FormEvent) {
    e.preventDefault();
    if (name.trim().length < 2) {
      setEditingAvatar(false);
      setError('نام نمایشی باید حداقل ۲ کاراکتر باشد.');
      return;
    }
    setError('');
    setBusy(true);
    try {
      onSave(
        await api<User>('/me', 'PATCH', { name, role, avatar: color, avatar_config: appearance }),
      );
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={
        editingAvatar
          ? 'شخصیت خودت را بساز'
          : user.name
            ? 'پروفایل شما'
            : 'قبل از شروع، همدیگر را بشناسیم'
      }
      wide={editingAvatar}
      onClose={onClose}
    >
      {!editingAvatar && (
        <div className="profile-character-summary">
          <Avatar name={name || 'شما'} color={color} config={appearance} />
          <div>
            <strong>یک چهره برای دیدگاه تو</strong>
            <p>آواتارت زیر کارت و کنار اسم تو دیده می‌شود.</p>
            <button type="button" className="text-button" onClick={() => setEditingAvatar(true)}>
              <Sparkles size={16} />
              شخصیتت را بساز
              <ChevronLeft size={16} />
            </button>
          </div>
        </div>
      )}
      <form onSubmit={save}>
        {editingAvatar ? (
          <AvatarBuilder
            value={appearance}
            background={color}
            onChange={setAppearance}
            onBackground={setColor}
          />
        ) : (
          <>
            <label>
              نام نمایشی
              <input
                aria-label="نام نمایشی"
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
                minLength={2}
                maxLength={50}
                placeholder="مثلاً پارسا احمدی"
                required
              />
            </label>
            <label>
              نقش شما در تیم <span className="muted">(اختیاری)</span>
              <input
                value={role}
                onChange={(e) => setRole(e.target.value)}
                maxLength={60}
                placeholder="مثلاً توسعه‌دهنده بک‌اند"
              />
            </label>
            <label>
              شماره موبایل
              <input value={user.phone} dir="ltr" readOnly className="readonly" />
            </label>
          </>
        )}
        {error && <p className="form-error">{error}</p>}
        <div className="profile-save-actions">
          {editingAvatar && (
            <button
              type="button"
              aria-label="بازگشت به پروفایل"
              className="secondary"
              onClick={() => setEditingAvatar(false)}
            >
              <ArrowRight size={16} />
              پروفایل
            </button>
          )}
          <Button busy={busy} className="primary full">
            ذخیره پروفایل
            <Check size={17} />
          </Button>
        </div>
      </form>
    </Modal>
  );
}
function CreateModal({ onClose, onCreate }: { onClose: () => void; onCreate: (r: Room) => void }) {
  const [name, setName] = useState(''),
    [deck, setDeck] = useState('fibonacci'),
    [theme, setTheme] = useState<RoomTheme>('violet'),
    [visibility, setVisibility] = useState<'private' | 'public'>('private'),
    [busy, setBusy] = useState(false),
    [error, setError] = useState('');
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      onCreate(await api<Room>('/rooms', 'POST', { name, deck, theme, visibility }));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title="یک میز برای تیم شما" onClose={onClose}>
      <p className="muted">دسترسی میز را انتخاب کنید و تخمین بعدی را با تیم شروع کنید.</p>
      <form onSubmit={submit}>
        <label>
          نام جلسه
          <input
            autoFocus
            aria-label="نام جلسه"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="مثلاً برنامه‌ریزی اسپرینت ۲۴"
            minLength={2}
            maxLength={80}
            required
          />
        </label>
        <label>دسته کارت</label>
        <div className="deck-options">
          {[
            ['fibonacci', 'فیبوناچی', '0, 1, 2, 3, 5, 8, 13, 21…'],
            ['tshirt', 'اندازه تی‌شرت', 'XS, S, M, L, XL'],
          ].map(([id, title, desc]) => (
            <button
              type="button"
              className={deck === id ? 'selected' : ''}
              onClick={() => setDeck(id)}
              key={id}
            >
              <span>
                {title}
                {deck === id && <CheckCircle2 size={18} />}
              </span>
              <small dir="ltr">{desc}</small>
            </button>
          ))}
        </div>
        <VisibilityPicker value={visibility} onChange={setVisibility} disabled={busy} />
        <ThemePicker value={theme} onChange={setTheme} disabled={busy} />
        {error && <p className="form-error">{error}</p>}
        <Button busy={busy} className="primary full">
          ساخت اتاق
          <Plus size={18} />
        </Button>
      </form>
    </Modal>
  );
}
function Landing({
  onLogin,
  onDemo,
  onJoin,
}: {
  onLogin: () => void;
  onDemo: () => void;
  onJoin: (id: string) => void;
}) {
  return (
    <main className="landing">
      <div className="landing-hero">
        <div className="hero-content">
          <span className="eyebrow">
            <span className="live-dot" />
            برای تیم‌هایی که با هم می‌سازند
          </span>
          <h1>
            هر کارت، یک دیدگاه.
            <br />
            <span>با هم بهتر تخمین بزنیم.</span>
          </h1>
          <p>
            بحث‌های خوب، تخمین‌های بهتر. تیم را دور یک میز جمع کنید، مستقل رأی بدهید و با هم به یک
            تصمیم برسید.
          </p>
          <div className="hero-actions">
            <button className="primary" onClick={onLogin}>
              شروع یک جلسه
              <ArrowLeft size={19} />
            </button>
            <button className="secondary" onClick={onDemo}>
              <Play size={17} />
              امتحان اتاق نمونه
            </button>
          </div>
          <div className="hero-proof">
            <div className="stacked-avatars">
              <Avatar name="سارا" color="peach" small />
              <Avatar name="علی" color="blue" small />
              <Avatar name="نیکا" color="mint" small />
            </div>
            <span>بدون نصب · ورود با شماره موبایل · کاملاً زنده</span>
          </div>
        </div>
        <div className="hero-art" aria-hidden="true">
          <div className="art-grid" />
          <div className="floating-badge">
            <span className="live-dot" />
            همه آماده‌اند!
          </div>
          <div className="hero-card back-card">
            ۸<span>STORY POINTS</span>
          </div>
          <div className="hero-card front-card">
            ۵<span>STORY POINTS</span>
            <Sparkles size={25} />
          </div>
          <div className="hero-art-bottom">
            <Avatar name="سارا" color="peach" />
            <Avatar name="پارسا" color="violet" />
            <Avatar name="علی" color="mint" />
            <span>
              یک تیم، یک تصمیم
              <Heart size={14} />
            </span>
          </div>
          <div className="art-star">✳</div>
        </div>
      </div>
      <PublicRooms onJoin={onJoin} />
      <section className="steps-section">
        <div className="section-intro">
          <span className="eyebrow">کمتر از یک دقیقه تا اولین رأی</span>
          <h2>ساده شروع کنید. با هم جلو بروید.</h2>
        </div>
        <div className="steps-grid">
          {[
            [Users, 'تیم را دعوت کنید', 'یک اتاق بسازید و لینک دعوت را برای هم‌تیمی‌ها بفرستید.'],
            [
              EyeOff,
              'مستقل رأی بدهید',
              'هر کس کارت خودش را انتخاب می‌کند؛ بدون تأثیر از رأی دیگران.',
            ],
            [
              Sparkles,
              'با هم تصمیم بگیرید',
              'گرداننده کارت‌ها را رو می‌کند و تیم به تخمین نهایی می‌رسد.',
            ],
          ].map(([Icon, title, desc], i) => {
            const I = Icon as typeof Users;
            return (
              <article key={i}>
                <span className="step-number">0{i + 1}</span>
                <div className="step-icon">
                  <I size={23} />
                </div>
                <h3>{title as string}</h3>
                <p>{desc as string}</p>
              </article>
            );
          })}
        </div>
      </section>
      <footer>
        با فکرهای متفاوت، به یک تخمین مشترک برسید.<span>هم‌تخمین · Planning Poker</span>
      </footer>
    </main>
  );
}
function Dashboard({
  user,
  onCreate,
  onRooms,
}: {
  user: User;
  onCreate: () => void;
  onRooms: () => void;
}) {
  return (
    <main className="dashboard">
      <div className="dashboard-title">
        <div>
          <span className="eyebrow">فضای کاری شما</span>
          <h1>
            سلام {user.name.split(' ')[0]}، آماده تخمین هستی؟ <span>☀</span>
          </h1>
          <p className="muted">یک جلسه جدید شروع کن یا به میز تیمت برگرد.</p>
        </div>
        <button className="primary" onClick={onCreate}>
          <Plus size={19} /> جلسه جدید
        </button>
      </div>
      <div className="dashboard-banner">
        <div>
          <span className="banner-label">
            <Sparkles size={16} /> یک تصمیم بهتر از چند دیدگاه شروع می‌شود
          </span>
          <h2>تخمین بعدی را با هم بسازید.</h2>
          <p>لینک اتاق را بفرستید؛ بقیه‌اش فقط یک کارت فاصله دارد.</p>
          <button onClick={onCreate}>
            ساخت اتاق <ArrowLeft size={17} />
          </button>
        </div>
        <div className="banner-cards" aria-hidden="true">
          <span>۳</span>
          <span>۵</span>
          <span>۸</span>
        </div>
      </div>
      <div className="dashboard-room-hint">
        <span className="empty-icon">
          <LayoutGrid size={28} />
        </span>
        <h2>یک اتاق، یک گفت‌وگوی تازه</h2>
        <p className="muted">
          از کارت اتاق‌ها، جلسهٔ خودت را انتخاب کن یا به یک اتاق عمومی بپیوند.
        </p>
        <button className="secondary" onClick={onRooms}>
          <LayoutGrid size={17} /> نمایش اتاق‌ها
        </button>
      </div>
    </main>
  );
}
function RoomScreen({
  room,
  user,
  demo,
  connected,
  busy,
  onCommand,
  onBack,
  onInvite,
  onSettings,
  onThemes,
  onHelp,
  notify,
}: {
  room: Room;
  user: User;
  demo: boolean;
  connected: boolean;
  busy: boolean;
  onCommand: (c: Command) => Promise<void>;
  onBack: () => void;
  onInvite: () => void;
  onSettings: () => void;
  onThemes: () => void;
  onHelp: () => void;
  notify: (s: string) => void;
}) {
  const [search, setSearch] = useState(''),
    [add, setAdd] = useState(false),
    [title, setTitle] = useState(''),
    [desc, setDesc] = useState(''),
    [detail, setDetail] = useState<Round | null>(null),
    [estimate, setEstimate] = useState('5'),
    [elapsed, setElapsed] = useState(0),
    [showPeople, setShowPeople] = useState(false),
    [showTasks, setShowTasks] = useState(false),
    [resultsCollapsed, setResultsCollapsed] = useState(false),
    [tableCollapsed, setTableCollapsed] = useState(false);
  const [celebration, setCelebration] = useState<{ attempt: string; value: string } | null>(null);
  const consensus = consensusState(room);
  const previousConsensus = useRef(consensus);
  const unanimous = consensus.value !== null;
  const tableCollapseRef = useRef<HTMLButtonElement>(null);
  const tableExpandRef = useRef<HTMLButtonElement>(null);
  const taskListRef = useRef<HTMLDivElement>(null);
  const previousTableCollapsed = useRef(tableCollapsed);
  const host = room.host_id === user.id,
    active = room.rounds.find((r) => r.id === room.active_round_id),
    revealed = active?.status === 'revealed' || active?.status === 'completed',
    voting = active?.status === 'voting',
    me = room.members.find((m) => m.id === user.id),
    members = room.members.filter((m) => room.closed || m.online),
    voters = members.filter((m) => !m.observer),
    voted = voters.filter((m) => m.has_voted).length,
    resultVoters =
      active?.status === 'completed' || room.closed
        ? room.members.filter((m) => !m.observer)
        : voters,
    resultVoted = resultVoters.filter((m) => m.has_voted).length,
    online = room.members.filter((m) => m.online).length;
  useEffect(() => {
    const tick = () => {
      setElapsed(
        active?.started_at && voting
          ? Math.max(0, Math.floor((Date.now() - new Date(active.started_at).getTime()) / 1000))
          : 0,
      );
    };
    tick();
    const timer = setInterval(tick, 1000);
    return () => clearInterval(timer);
  }, [active?.started_at, voting]);
  useEffect(() => {
    setEstimate(room.cards.includes('5') ? '5' : room.cards[0]);
  }, [room.deck, room.active_round_id]);
  useEffect(() => {
    setSearch('');
    taskListRef.current?.scrollTo({ top: 0 });
  }, [room.active_round_id]);
  useEffect(() => {
    setResultsCollapsed(false);
  }, [room.active_round_id, revealed]);
  useEffect(() => {
    if (previousTableCollapsed.current !== tableCollapsed)
      (tableCollapsed ? tableExpandRef : tableCollapseRef).current?.focus();
    previousTableCollapsed.current = tableCollapsed;
  }, [tableCollapsed]);
  useEffect(() => {
    const previous = previousConsensus.current;
    previousConsensus.current = consensus;
    if (!consensus.revealed) setCelebration(null);
    else if (shouldCelebrate(previous, consensus))
      setCelebration({ attempt: consensus.attempt, value: consensus.value! });
  }, [consensus.attempt, consensus.revealed, consensus.value]);
  const visibleRounds = room.rounds
    .map((round, index) => ({ round, number: index + 1 }))
    .filter(({ round }) => round.title.includes(search))
    .sort(({ round: a }, { round: b }) =>
      a.id === room.active_round_id ? -1 : b.id === room.active_round_id ? 1 : 0,
    );
  async function addTask(e: FormEvent) {
    e.preventDefault();
    try {
      await onCommand({ action: 'add_round', title, description: desc });
      setTitle('');
      setDesc('');
      setAdd(false);
    } catch {}
  }
  function exportCSV() {
    const quote = (s: string) =>
      '"' + (/^[=+@\-\t\r]/.test(s) ? "'" : '') + s.replace(/"/g, '""') + '"';
    const csv =
      '\ufeff' +
      [
        ['عنوان', 'توضیحات', 'وضعیت', 'امتیاز'],
        ...room.rounds.map((r) => [r.title, r.description, statusText[r.status], r.estimate]),
      ]
        .map((row) => row.map(quote).join(','))
        .join('\r\n');
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
    const a = document.createElement('a');
    a.href = url;
    a.download = 'planning-poker.csv';
    a.click();
    URL.revokeObjectURL(url);
    notify('خروجی جلسه دانلود شد.');
  }
  const seats = members
    .filter((m) => !m.observer)
    .sort((a, b) => (a.id === user.id ? -1 : b.id === user.id ? 1 : 0))
    .slice(0, 8);
  const seatPositions = [
    [50, 112],
    [20, -6],
    [50, -12],
    [80, -6],
    [80, 106],
    [20, 106],
    [94, 53],
    [6, 53],
  ];
  return (
    <main className="room-screen">
      {celebration && (
        <ConsensusCelebration
          key={celebration.attempt}
          value={celebration.value}
          onDone={() => setCelebration(null)}
        />
      )}
      <div className="room-heading">
        <div className="room-heading-info">
          <button className="icon-button back" onClick={onBack} aria-label="بازگشت به جلسه‌ها">
            <ArrowRight size={21} />
          </button>
          <div>
            <div className="room-name">
              <h1>{room.name}</h1>
              <span className={`badge ${demo ? 'orange' : room.closed ? 'gray' : 'green'}`}>
                {demo ? 'اتاق نمونه' : room.closed ? 'پایان‌یافته' : 'جلسه زنده'}
              </span>
            </div>
            <p>
              <span className={`live-dot ${connected ? '' : 'offline'}`} />
              {demo
                ? 'برای آشنایی؛ اعضا و رأی‌ها شبیه‌سازی شده‌اند'
                : room.closed
                  ? 'جلسه پایان یافته · اتصال زنده بسته است'
                  : connected
                    ? 'متصل · همه تغییرات هم‌زمان نمایش داده می‌شود'
                    : 'در حال اتصال دوباره…'}
              <span className="heading-separator">/</span>
              {room.deck === 'fibonacci' ? 'فیبوناچی' : 'اندازه تی‌شرت'}
            </p>
          </div>
        </div>
        <div className="room-actions">
          <button className="secondary small-button" onClick={onInvite}>
            <Users size={17} />
            دعوت اعضا
          </button>
          <button
            className="icon-button"
            aria-label="خروجی جلسه"
            title="خروجی CSV"
            onClick={exportCSV}
          >
            <Download size={19} />
          </button>
          {host && (
            <button
              className="icon-button room-color-button"
              aria-label="تغییر رنگ تم اتاق"
              title="تغییر رنگ تم اتاق"
              onClick={onThemes}
            >
              <Palette size={19} />
            </button>
          )}
          <button className="icon-button" aria-label="تنظیمات جلسه" onClick={onSettings}>
            <Settings2 size={19} />
          </button>
        </div>
      </div>
      <div className="mobile-panel-tabs">
        <button
          className={showPeople ? 'selected' : ''}
          aria-expanded={showPeople}
          aria-controls="room-people-panel"
          onClick={() => {
            setShowPeople(!showPeople);
            setShowTasks(false);
          }}
        >
          <Users size={16} />
          اعضا ({fa(members.length)})
        </button>
        <button
          className={showTasks ? 'selected' : ''}
          aria-expanded={showTasks}
          aria-controls="room-tasks-panel"
          onClick={() => {
            setShowTasks(!showTasks);
            setShowPeople(false);
          }}
        >
          <ListTodo size={16} />
          تسک‌ها ({fa(room.rounds.length)})
        </button>
      </div>
      <div className={`room-layout ${resultsCollapsed ? 'results-collapsed' : ''}`}>
        <div className="session-sidebar">
          <aside
            id="room-people-panel"
            className={`people-panel ${showPeople ? 'mobile-show' : ''}`}
          >
            <div className="panel-title">
              <h2>
                <Users size={18} />
                اعضای جلسه<span>{fa(members.length)}</span>
              </h2>
              <span className="online-count">{fa(online)} آنلاین</span>
            </div>
            <div
              className="participant-list"
              tabIndex={0}
              role="region"
              aria-label="فهرست اعضای جلسه"
            >
              {members.map((m) => (
                <div className={`participant ${m.id === user.id ? 'is-me' : ''}`} key={m.id}>
                  <div className="avatar-status">
                    <Avatar name={m.name} color={m.avatar} config={m.avatar_config} />
                    <i className={m.online ? 'online' : ''} />
                  </div>
                  <div className="participant-text">
                    <strong>
                      {m.name}
                      {m.id === user.id && <small> (شما)</small>}
                    </strong>
                    <span>
                      {m.id === room.host_id ? (
                        <>
                          <Crown size={12} />
                          گرداننده
                        </>
                      ) : m.observer ? (
                        <>
                          <Eye size={12} />
                          ناظر
                        </>
                      ) : (
                        m.role || 'عضو تیم'
                      )}
                    </span>
                  </div>
                  {m.observer ? (
                    <Eye size={16} className="muted" />
                  ) : revealed && m.vote ? (
                    <span className="vote-chip">{m.vote}</span>
                  ) : m.has_voted ? (
                    <span className="voted-check" title="رأی داده">
                      <Check size={15} />
                    </span>
                  ) : (
                    <CircleDashed size={18} className="waiting-icon" />
                  )}
                </div>
              ))}
            </div>
          </aside>
          <aside id="room-tasks-panel" className={`tasks-panel ${showTasks ? 'mobile-show' : ''}`}>
            <div className="panel-title">
              <h2>
                <ListTodo size={18} />
                تسک‌ها<span>{fa(room.rounds.length)}</span>
              </h2>
              <button className="icon-button" onClick={exportCSV} aria-label="دانلود تسک‌ها">
                <Download size={16} />
              </button>
            </div>
            <div className="task-search search">
              <Search size={16} />
              <input
                aria-label="جستجوی تسک"
                placeholder="جستجوی تسک…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <div
              ref={taskListRef}
              className="task-list"
              tabIndex={0}
              role="region"
              aria-label="فهرست تسک‌های جلسه"
            >
              {visibleRounds.map(({ round: r, number }) => (
                <button
                  className={`task-card ${r.id === room.active_round_id ? 'current' : ''} ${r.status === 'completed' ? 'done' : ''}`}
                  key={r.id}
                  aria-current={r.id === room.active_round_id ? 'step' : undefined}
                  onClick={() => setDetail(r)}
                >
                  <div className="task-card-top">
                    <span className={`task-status ${r.status}`}>
                      {r.id === room.active_round_id && voting ? (
                        <span className="live-dot" />
                      ) : r.status === 'completed' ? (
                        <Check size={12} />
                      ) : null}
                      {statusText[r.status]}
                    </span>
                    <span className="task-id" dir="ltr">
                      PP-{number}
                    </span>
                  </div>
                  <h3>{r.title}</h3>
                  {r.id === room.active_round_id && r.description && (
                    <p className="task-active-description">{r.description}</p>
                  )}
                  <div className="task-card-footer">
                    <span>
                      {r.id === room.active_round_id ? (
                        <>
                          <Users size={13} />
                          {fa(voted)} / {fa(voters.length)} رأی
                        </>
                      ) : r.status === 'completed' ? (
                        'تخمین نهایی'
                      ) : (
                        'آماده تخمین'
                      )}
                    </span>
                    {r.id === room.active_round_id && voting && (
                      <span className="task-current-timer" title="زمان رأی‌گیری">
                        <Clock3 size={12} />
                        <span dir="ltr">
                          {String(Math.floor(elapsed / 60)).padStart(2, '0')}:
                          {String(elapsed % 60).padStart(2, '0')}
                        </span>
                      </span>
                    )}
                    {r.estimate ? (
                      <span className="estimate-badge">{r.estimate}</span>
                    ) : (
                      <ChevronLeft size={16} />
                    )}
                  </div>
                </button>
              ))}
              {!visibleRounds.length && (
                <div className="empty-tasks">
                  <ListTodo size={28} />
                  <p>{search ? 'تسکی پیدا نشد' : 'هنوز تسکی اضافه نشده'}</p>
                  <small>
                    {search
                      ? 'عبارت جستجو را تغییر دهید.'
                      : 'یک موضوع کوچک، شروع یک گفت‌وگوی خوب است.'}
                  </small>
                </div>
              )}
            </div>
            {host && !room.closed && (
              <button className="add-task" onClick={() => setAdd(true)}>
                <Plus size={18} />
                اضافه کردن تسک
              </button>
            )}
            <div className="session-summary">
              <span>پیشرفت جلسه</span>
              <strong>
                {fa(room.rounds.filter((r) => r.status === 'completed').length)} از{' '}
                {fa(room.rounds.length)} تسک
              </strong>
              <div className="progress-track">
                <i
                  style={{
                    width: `${room.rounds.length ? (room.rounds.filter((r) => r.status === 'completed').length / room.rounds.length) * 100 : 0}%`,
                  }}
                />
              </div>
            </div>
          </aside>
        </div>
        <section className="play-area" aria-label="رأی‌گیری و میز گفت‌وگو">
          <div
            className="discussion-table"
            hidden={tableCollapsed}
            tabIndex={0}
            role="region"
            aria-label="میز گفت‌وگو"
          >
            <div className="table-toolbar">
              <h2>
                <Users size={17} /> میز گفت‌وگو
              </h2>
              <button
                ref={tableCollapseRef}
                className="icon-button"
                aria-label="جمع کردن میز گفت‌وگو"
                aria-expanded={true}
                aria-controls="discussion-table-content"
                title="جمع کردن میز گفت‌وگو"
                onClick={() => setTableCollapsed(true)}
              >
                <Minimize2 size={17} />
              </button>
            </div>
            <div id="discussion-table-content">
              <div className={`table-stage ${revealed ? 'revealed' : ''}`}>
                <div className="table-oval">
                  <div className="table-oval-inner">
                    <span className="table-spark">
                      <Sparkles size={24} />
                    </span>
                    <h3>
                      {room.closed
                        ? 'جلسه به پایان رسید'
                        : revealed
                          ? unanimous
                            ? 'هم‌نظر هستید!'
                            : 'وقت گفت‌وگوست'
                          : active
                            ? 'هر کارت، یک دیدگاه'
                            : 'تیم اینجا دور هم جمع می‌شود'}
                    </h3>
                    <p>
                      {room.closed
                        ? 'خروجی تخمین‌ها آماده دریافت است.'
                        : revealed
                          ? 'دیدگاه‌ها را مقایسه کنید و به یک تخمین مشترک برسید.'
                          : active
                            ? 'کارتت را انتخاب کن؛ بقیه هم در حال فکر کردن‌اند.'
                            : 'تسک اول را اضافه کنید تا تخمین شروع شود.'}
                    </p>
                    {voting && (
                      <div className="table-vote-progress">
                        <span>
                          {fa(voted)} از {fa(voters.length)} نفر رأی داده‌اند
                        </span>
                        <div className="progress-track">
                          <i
                            style={{
                              width: `${voters.length ? (voted / voters.length) * 100 : 0}%`,
                            }}
                          />
                        </div>
                      </div>
                    )}
                    {host && voting && (
                      <Button
                        busy={busy}
                        disabled={!voted || room.closed}
                        className="primary reveal-button"
                        onClick={() => void onCommand({ action: 'reveal' }).catch(() => {})}
                      >
                        <Eye size={17} />
                        رو کردن کارت‌ها
                      </Button>
                    )}
                    {!host && voting && (
                      <span className="host-wait">منتظر نمایش کارت‌ها توسط گرداننده</span>
                    )}
                    {revealed && (
                      <span className="table-result">
                        {active?.status === 'completed'
                          ? `تخمین نهایی: ${active.estimate}`
                          : room.deck === 'fibonacci'
                            ? Object.keys(room.distribution).some((v) => v !== '?' && v !== '☕')
                              ? `میانگین: ${room.average.toLocaleString('fa-IR', { maximumFractionDigits: 1 })}`
                              : 'رأی عددی ثبت نشده'
                            : 'کارت‌ها رو شدند'}
                      </span>
                    )}
                  </div>
                </div>
                {seats.map((m, i) => {
                  let pos = seatPositions[i];
                  if (seats.length <= 6) {
                    pos = [
                      [50, 112],
                      [22, -4],
                      [50, -10],
                      [78, -4],
                      [22, 106],
                      [78, 106],
                    ][i];
                  }
                  return (
                    <div
                      key={m.id}
                      className={`table-seat ${m.id === user.id ? 'self-seat' : ''}`}
                      style={{ left: `${pos[0]}%`, top: `${pos[1]}%` }}
                    >
                      <div
                        className={`mini-card ${revealed && m.vote ? 'flipped' : m.has_voted ? 'card-ready' : 'card-waiting'}`}
                      >
                        {revealed ? (
                          m.vote || '—'
                        ) : m.has_voted ? (
                          <span className="card-back-diamond">✦</span>
                        ) : (
                          <span>···</span>
                        )}
                      </div>
                      <div className="seat-person">
                        <Avatar name={m.name} color={m.avatar} config={m.avatar_config} small />
                        <span className="seat-name" title={m.name}>
                          {m.id === user.id ? 'شما' : m.name.split(' ')[0]}
                          {m.id === room.host_id && <Crown size={10} />}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
          {tableCollapsed && host && voting && !room.closed && (
            <Button
              className="primary table-collapsed-reveal"
              busy={busy}
              disabled={!voted}
              onClick={() => void onCommand({ action: 'reveal' }).catch(() => {})}
            >
              <Eye size={17} /> رو کردن کارت‌ها · {fa(voted)} از {fa(voters.length)} رأی
            </Button>
          )}
          <div className="deck-area">
            <div className="deck-heading">
              <h3>
                {room.closed
                  ? 'جلسه تمام شده است'
                  : me?.observer
                    ? 'شما ناظر این جلسه هستید'
                    : revealed
                      ? 'کارت‌ها رو شدند؛ با تیم گفت‌وگو کنید'
                      : 'کارتت را انتخاب کن'}
              </h3>
              {voting && room.my_vote ? (
                <span>
                  <CheckCircle2 size={14} />
                  رأی شما ثبت شد؛ می‌توانید تغییرش دهید.
                </span>
              ) : (
                <span>
                  {room.deck === 'fibonacci'
                    ? 'امتیاز بر اساس پیچیدگی، نه زمان'
                    : 'اندازه نسبی پیچیدگی تسک'}
                </span>
              )}
            </div>
            <div className="voting-deck" dir="ltr">
              {room.cards.map((card) => (
                <button
                  key={card}
                  className={`voting-card ${room.my_vote === card ? 'picked' : ''}`}
                  aria-label={`رأی ${card}`}
                  aria-pressed={room.my_vote === card}
                  disabled={!voting || me?.observer || room.closed || busy}
                  onClick={() =>
                    void onCommand({
                      action: 'vote',
                      value: room.my_vote === card ? '' : card,
                      round_id: room.active_round_id,
                    }).catch(() => {})
                  }
                >
                  <span className="corner-number">{card === '☕' ? '' : card}</span>
                  <span className="card-number">{card === '☕' ? <Coffee size={24} /> : card}</span>
                  <span className="card-diamond">✦</span>
                </button>
              ))}
            </div>
            <div className="deck-footer">
              <span>
                <EyeOff size={14} />
                {revealed
                  ? 'همه رأی‌ها برای اعضای جلسه قابل مشاهده‌اند.'
                  : 'کارت شما فقط برای خودتان قابل مشاهده است.'}
              </span>
              {host && !room.closed && active && active.status !== 'completed' && (
                <button
                  className="text-button"
                  disabled={busy}
                  onClick={() => void onCommand({ action: 'reset' }).catch(() => {})}
                >
                  <RotateCcw size={14} />
                  شروع دوباره
                </button>
              )}
            </div>
          </div>
        </section>
        <aside
          className={`results-panel ${resultsCollapsed ? 'is-collapsed' : ''}`}
          aria-labelledby="results-panel-title"
        >
          <div className="results-panel-heading">
            <h2 id="results-panel-title">
              <ChartNoAxesColumn size={18} /> نتایج رأی‌گیری
            </h2>
            <button
              className="icon-button"
              aria-label={resultsCollapsed ? 'باز کردن نتایج رأی‌گیری' : 'جمع کردن نتایج رأی‌گیری'}
              aria-expanded={!resultsCollapsed}
              aria-controls="results-panel-content"
              title={resultsCollapsed ? 'باز کردن نتایج رأی‌گیری' : 'جمع کردن نتایج رأی‌گیری'}
              onClick={() => setResultsCollapsed(!resultsCollapsed)}
            >
              {resultsCollapsed ? <ChartNoAxesColumn size={20} /> : <ChevronUp size={17} />}
            </button>
          </div>
          {!resultsCollapsed && (
            <div id="results-panel-content" className="results-panel-content">
              {revealed ? (
                <>
                  <div className="results-summary">
                    <span>
                      {active?.status === 'completed'
                        ? 'تخمین نهایی'
                        : unanimous
                          ? 'تیم هم‌نظر است'
                          : room.deck === 'fibonacci'
                            ? 'میانگین امتیازها'
                            : 'کارت‌ها رو شدند'}
                    </span>
                    {active?.status === 'completed' ? (
                      <strong>{active.estimate}</strong>
                    ) : room.deck === 'fibonacci' &&
                      Object.keys(room.distribution).some((value) =>
                        Number.isFinite(Number(value)),
                      ) ? (
                      <strong>
                        {room.average.toLocaleString('fa-IR', { maximumFractionDigits: 1 })}
                      </strong>
                    ) : (
                      <Eye size={28} />
                    )}
                    <small>
                      {fa(resultVoted)} از {fa(resultVoters.length)} نفر رأی داده‌اند
                    </small>
                  </div>
                  <h3 className="distribution-title">توزیع امتیازها</h3>
                  <div className="results-strip">
                    <div className="distribution">
                      {Object.entries(room.distribution).map(([v, n]) => (
                        <div key={v}>
                          <span className="result-card">{v}</span>
                          <small>{fa(n)} رأی</small>
                        </div>
                      ))}
                    </div>
                    {host && !room.closed && active?.status === 'revealed' ? (
                      <div className="final-estimate">
                        <label htmlFor="final-estimate">تخمین نهایی</label>
                        <select
                          id="final-estimate"
                          value={estimate}
                          onChange={(e) => setEstimate(e.target.value)}
                        >
                          {room.cards
                            .filter((c) => c !== '?' && c !== '☕')
                            .map((c) => (
                              <option key={c}>{c}</option>
                            ))}
                        </select>
                        <Button
                          className="primary small-button"
                          busy={busy}
                          onClick={() =>
                            void onCommand({ action: 'estimate', value: estimate }).catch(() => {})
                          }
                        >
                          <Check size={16} />
                          ثبت
                        </Button>
                      </div>
                    ) : active?.status === 'completed' ? (
                      <span className="completed-result">
                        <CheckCircle2 size={18} />
                        تخمین ثبت شد · {active.estimate}
                      </span>
                    ) : null}
                  </div>
                </>
              ) : (
                <div className="results-empty">
                  <EyeOff size={30} />
                  <h3>امتیازها هنوز پنهان‌اند</h3>
                  <p>بعد از رو شدن کارت‌ها، نتیجه و توزیع رأی‌ها را اینجا می‌بینید.</p>
                </div>
              )}
            </div>
          )}
        </aside>
      </div>
      {tableCollapsed && (
        <button
          ref={tableExpandRef}
          className="workspace-floating-button table-floating-button"
          aria-label="باز کردن میز گفت‌وگو"
          aria-expanded={false}
          aria-controls="discussion-table-content"
          title="میز گفت‌وگو"
          onClick={() => setTableCollapsed(false)}
        >
          <Users size={23} />
          <span className="floating-count">{fa(voters.length)}</span>
        </button>
      )}
      <div className="room-bottom">
        <span>
          <ShieldCheck size={13} />
          رأی‌گیری مستقل و امن
        </span>
        <span>فکرهای متفاوت. یک تخمین مشترک.</span>
        <button className="text-button" onClick={onHelp}>
          راهنما
          <CircleHelp size={13} />
        </button>
      </div>
      {add && (
        <Modal title="تسک جدید" onClose={() => setAdd(false)}>
          <form onSubmit={addTask}>
            <label>
              عنوان تسک
              <input
                autoFocus
                aria-label="عنوان تسک"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                minLength={2}
                maxLength={160}
                placeholder="چه چیزی را می‌خواهید تخمین بزنید؟"
                required
              />
            </label>
            <label>
              توضیحات
              <textarea
                aria-label="توضیحات تسک"
                value={desc}
                onChange={(e) => setDesc(e.target.value)}
                maxLength={3000}
                placeholder="معیار پذیرش یا جزئیاتی که به تیم کمک می‌کند…"
                rows={4}
              />
            </label>
            <Button className="primary full" busy={busy}>
              اضافه کردن تسک
              <Plus size={18} />
            </Button>
          </form>
        </Modal>
      )}
      {detail && (
        <Modal title={detail.title} onClose={() => setDetail(null)}>
          <span className={`task-status ${detail.status}`}>{statusText[detail.status]}</span>
          <p className="detail-description">
            {detail.description || 'توضیحی برای این تسک ثبت نشده است.'}
          </p>
          {detail.estimate && (
            <div className="detail-estimate">
              <CheckCircle2 size={18} />
              تخمین نهایی: <b>{detail.estimate}</b>
            </div>
          )}
          {host && detail.status === 'pending' && !room.closed && (
            <Button
              busy={busy}
              className="primary full"
              disabled={!!active && active.status !== 'completed'}
              onClick={() =>
                void onCommand({ action: 'start_round', round_id: detail.id })
                  .then(() => setDetail(null))
                  .catch(() => {})
              }
            >
              <Play size={17} />
              شروع رأی‌گیری این تسک
            </Button>
          )}
          {host && detail.status === 'pending' && active && active.status !== 'completed' && (
            <p className="muted">ابتدا تخمین تسک جاری را ثبت کنید.</p>
          )}
        </Modal>
      )}
    </main>
  );
}
export default function App() {
  const [path, setPath] = useState(location.pathname),
    [user, setUser] = useState<User | null>(null),
    [authReady, setAuthReady] = useState(false),
    [modal, setModal] = useState(''),
    [room, setRoom] = useState<Room | null>(null),
    [roomError, setRoomError] = useState(''),
    [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('connecting'),
    [disconnectReason, setDisconnectReason] = useState(''),
    [reconnect, setReconnect] = useState(0),
    [busy, setBusy] = useState(false),
    [toast, setToast] = useState(''),
    [dark, setDark] = useState(false),
    [roomsCollapsed, setRoomsCollapsed] = useState(
      () => window.matchMedia('(max-width: 1279px)').matches,
    );
  const eventsRef = useRef<EventSource | null>(null);
  const demo = path === '/demo',
    roomID = path.startsWith('/room/') ? path.split('/')[2] : '',
    shownUser = demo ? demoUser : user;
  const notify = (message: string) => setToast(message);
  function navigate(p: string) {
    history.pushState({}, '', p);
    setPath(p);
    setRoomError('');
  }
  useEffect(() => {
    const pop = () => setPath(location.pathname);
    window.addEventListener('popstate', pop);
    return () => window.removeEventListener('popstate', pop);
  }, []);
  useEffect(() => {
    let alive = true;
    api<User>('/me')
      .then((u) => {
        if (alive) {
          setUser(u);
          if (!u.name) setModal('profile');
        }
      })
      .catch((e) => {
        if (alive && !(e instanceof APIError && e.status === 401)) setToast(e.message);
      })
      .finally(() => {
        if (alive) setAuthReady(true);
      });
    return () => {
      alive = false;
    };
  }, []);
  useEffect(() => {
    document.documentElement.dataset.theme = dark ? 'dark' : 'light';
  }, [dark]);
  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => setToast(''), 5000);
    return () => clearTimeout(timer);
  }, [toast]);
  useEffect(() => {
    if (roomID && authReady && !user) setModal('auth');
  }, [roomID, authReady, user]);
  useEffect(() => {
    if (demo) {
      setRoom(demoRoom());
      setConnectionStatus('connected');
      return;
    }
    if (!roomID || !user?.name) {
      setRoom(null);
      return;
    }
    let alive = true;
    let events: EventSource | null = null;
    let stopStream: (() => void) | undefined;
    const disconnected = (reason: string) => {
      if (!alive) return;
      setConnectionStatus('disconnected');
      setDisconnectReason(reason);
      setModal((current) =>
        ['invite', 'settings', 'room-theme'].includes(current) ? '' : current,
      );
    };
    const offline = () => {
      stopStream?.();
      disconnected('اتصال اینترنت شما قطع شده است.');
    };
    const resume = () => {
      if (alive) setReconnect((value) => value + 1);
    };
    const pagehide = () => stopStream?.();
    const pageshow = (event: PageTransitionEvent) => {
      if (event.persisted) resume();
    };
    window.addEventListener('offline', offline);
    window.addEventListener('online', resume);
    window.addEventListener('pagehide', pagehide);
    window.addEventListener('pageshow', pageshow);
    setRoom((current) => (current?.id === roomID ? current : null));
    setRoomError('');
    setConnectionStatus('connecting');
    if (!navigator.onLine) offline();
    else
      api<Room>(`/rooms/${roomID}`)
        .catch((e) => {
          if (e instanceof APIError && e.status === 403)
            return api<Room>(`/rooms/${roomID}/join`, 'POST', { observer: false });
          throw e;
        })
        .then((r) => {
          if (!alive) return;
          setRoom(r);
          if (r.closed) return;
          if (!navigator.onLine) {
            offline();
            return;
          }
          events = new EventSource(`/api/rooms/${roomID}/events`);
          eventsRef.current = events;
          stopStream = openRoomConnection(events, user.id, {
            room: (next) => {
              if (!alive || next.id !== roomID) return;
              setRoom((old) => (!old || next.revision >= old.revision ? next : old));
              setConnectionStatus('connected');
            },
            disconnected,
            expired: () => {
              if (!alive) return;
              setConnectionStatus('expired');
              setDisconnectReason('نشست ورود شما پایان یافته است. برای بازگشت، دوباره وارد شوید.');
              setUser(null);
              setModal('auth');
            },
          });
        })
        .catch((e) => {
          if (!alive) return;
          if (e instanceof APIError && e.status === 401) {
            setConnectionStatus('expired');
            setDisconnectReason('برای بازگشت به اتاق، دوباره وارد شوید.');
            setUser(null);
            setModal('auth');
          } else if (e instanceof APIError && [403, 404, 409].includes(e.status)) {
            setRoomError(e.message);
          } else disconnected(e.message || 'ارتباط با اتاق برقرار نشد.');
        });
    return () => {
      alive = false;
      stopStream?.();
      events?.close();
      window.removeEventListener('offline', offline);
      window.removeEventListener('online', resume);
      window.removeEventListener('pagehide', pagehide);
      window.removeEventListener('pageshow', pageshow);
      if (eventsRef.current === events) eventsRef.current = null;
    };
  }, [roomID, user?.id, user?.name, demo, reconnect]);
  async function command(c: Command) {
    if (!room) return;
    if (demo) {
      setRoom((r) => demoCommand(r!, c));
      return;
    }
    if (!room.closed && connectionStatus !== 'connected') return;
    setBusy(true);
    try {
      const next = await api<Room>(`/rooms/${room.id}/commands`, 'POST', c);
      setRoom((old) => (!old || next.revision >= old.revision ? next : old));
    } catch (e) {
      notify((e as Error).message);
      throw e;
    } finally {
      setBusy(false);
    }
  }
  async function logout() {
    try {
      await api('/auth/logout', 'POST');
      setUser(null);
      setModal('');
      navigate('/');
      notify('از حساب خارج شدید.');
    } catch (e) {
      notify((e as Error).message);
    }
  }
  const needCreate = () => {
    if (!user) {
      setModal('auth');
    } else if (!user.name) {
      setModal('profile');
    } else setModal('create');
  };
  const closeModal = () => {
    if (modal === 'profile' && !user?.name) return;
    setModal('');
  };
  return (
    <div
      className={`app ${demo || roomID ? 'in-room' : ''} ${demo ? 'is-demo' : ''} ${roomsCollapsed ? '' : 'rooms-open'}`}
      data-room-theme={roomTheme(room?.theme)}
    >
      <header className="app-header">
        <button
          className="brand-button"
          onClick={() => navigate(user ? '/dashboard' : '/')}
          aria-label="صفحه اصلی"
        >
          <Brand />
        </button>
        <nav>
          {user ? (
            <button
              className={path === '/dashboard' ? 'nav-active' : ''}
              onClick={() => navigate('/dashboard')}
            >
              <LayoutGrid size={17} />
              جلسه‌های من
            </button>
          ) : (
            <span className="header-tagline">با هم فکر کنیم، بهتر بسازیم.</span>
          )}
          <button onClick={() => setModal('help')}>
            <CircleHelp size={17} />
            راهنما
          </button>
        </nav>
        <div className="header-actions">
          <button
            className="icon-button theme-button"
            aria-label="تغییر حالت روشن و تیره"
            title="حالت روشن و تیره برای نمایش شما"
            onClick={() => setDark(!dark)}
          >
            {dark ? '☀' : '☾'}
          </button>
          {user ? (
            <button
              className="profile-button"
              aria-label={`حساب کاربری ${user.name || 'شما'}`}
              onClick={() => setModal('account')}
            >
              <Avatar
                name={user.name || 'شما'}
                color={user.avatar}
                config={user.avatar_config}
                small
              />
              <span>{user.name || 'تکمیل پروفایل'}</span>
              <ChevronLeft size={14} />
            </button>
          ) : (
            <button className="secondary small-button" onClick={() => setModal('auth')}>
              <Smartphone size={16} />
              ورود / ثبت‌نام
            </button>
          )}
        </div>
      </header>
      {demo && (
        <div className="demo-banner">
          <Sparkles size={14} />
          <span>این یک اتاق نمونه است؛ تغییرات آن ذخیره نمی‌شود.</span>
          <button onClick={needCreate}>
            ساخت اتاق واقعی
            <ArrowLeft size={14} />
          </button>
        </div>
      )}
      <div className="workspace-body">
        {demo || roomID ? (
          !demo && !roomError && ['disconnected', 'expired'].includes(connectionStatus) ? (
            <main className="disconnected-room" aria-labelledby="disconnect-title">
              <div className="disconnect-card" role="status">
                <span className="disconnect-symbol">
                  <WifiOff size={34} />
                </span>
                <span className="eyebrow">{room?.name || 'اتاق شما'}</span>
                <h1 id="disconnect-title">ارتباط با جلسه قطع شد</h1>
                <p>{disconnectReason}</p>
                <p className="muted">پس از اتصال دوباره، به جمع اعضای حاضر برمی‌گردید.</p>
                <button
                  className="primary"
                  onClick={() => (user ? setReconnect((v) => v + 1) : setModal('auth'))}
                >
                  <RotateCcw size={18} />
                  {user ? 'اتصال دوباره به اتاق' : 'ورود دوباره'}
                </button>
                <button className="secondary" onClick={() => navigate(user ? '/dashboard' : '/')}>
                  بازگشت به جلسه‌ها
                </button>
              </div>
            </main>
          ) : room &&
            room.id === (demo ? room.id : roomID) &&
            shownUser &&
            (demo || room.closed || connectionStatus === 'connected') ? (
            <RoomScreen
              key={room.id}
              room={room}
              user={shownUser}
              demo={demo}
              connected={demo || connectionStatus === 'connected'}
              busy={busy}
              onCommand={command}
              onBack={() => navigate(user ? '/dashboard' : '/')}
              onInvite={() => setModal('invite')}
              onSettings={() => setModal('settings')}
              onThemes={() => setModal('room-theme')}
              onHelp={() => setModal('help')}
              notify={notify}
            />
          ) : (
            <div className="loading-room">
              {roomError ? (
                <>
                  <CircleHelp size={34} />
                  <h2>{roomError}</h2>
                  <button className="secondary" onClick={() => navigate('/dashboard')}>
                    بازگشت به جلسه‌ها
                  </button>
                </>
              ) : !user && authReady ? (
                <>
                  <Users size={36} />
                  <h2>تیمت منتظر توست</h2>
                  <p className="muted">برای پیوستن به اتاق، با شماره موبایل وارد شوید.</p>
                  <button className="primary" onClick={() => setModal('auth')}>
                    ورود به جلسه
                  </button>
                </>
              ) : (
                <>
                  <Loader2 className="spin" size={30} />
                  <p>در حال آماده کردن میز…</p>
                </>
              )}
            </div>
          )
        ) : path === '/dashboard' && user ? (
          <Dashboard user={user} onCreate={needCreate} onRooms={() => setRoomsCollapsed(false)} />
        ) : (
          <Landing
            onLogin={needCreate}
            onDemo={() => navigate('/demo')}
            onJoin={(id) => navigate(`/room/${id}`)}
          />
        )}
      </div>
      <RoomsDock
        key={`${user?.id || 'guest'}-${demo ? 'demo' : 'live'}`}
        user={user}
        room={demo || (roomID && (room?.closed || connectionStatus === 'connected')) ? room : null}
        demo={demo}
        collapsed={roomsCollapsed}
        onCollapse={setRoomsCollapsed}
        onCreate={needCreate}
        onOpen={(id) => {
          if (window.matchMedia('(max-width: 1279px)').matches) setRoomsCollapsed(true);
          const next = id === 'demo' ? '/demo' : `/room/${id}`;
          if (next !== path) navigate(next);
        }}
      />
      {modal === 'auth' && (
        <AuthModal
          onClose={() => setModal('')}
          onSuccess={(u) => {
            setUser(u);
            setModal(u.name ? '' : 'profile');
            if (!roomID && !demo) navigate('/dashboard');
          }}
        />
      )}
      {modal === 'profile' && user && (
        <ProfileModal
          user={user}
          onClose={closeModal}
          onSave={(u) => {
            setUser(u);
            setModal('');
            if (!roomID) navigate('/dashboard');
            notify('پروفایل شما ذخیره شد.');
          }}
        />
      )}
      {modal === 'create' && (
        <CreateModal
          onClose={closeModal}
          onCreate={(r) => {
            setModal('');
            navigate(`/room/${r.id}`);
            notify(
              r.visibility === 'public'
                ? 'اتاق عمومی ساخته شد و در صفحهٔ اصلی نمایش داده می‌شود.'
                : 'اتاق خصوصی ساخته شد. لینک دعوت را برای تیمت بفرست.',
            );
          }}
        />
      )}
      {modal === 'account' && user && (
        <Modal title="حساب شما" onClose={closeModal}>
          <div className="account-card">
            <Avatar name={user.name} color={user.avatar} config={user.avatar_config} />
            <div>
              <h3>{user.name}</h3>
              <span dir="ltr">{user.phone}</span>
            </div>
          </div>
          <button className="account-menu" onClick={() => setModal('profile')}>
            <Settings2 size={19} />
            ویرایش پروفایل
            <ChevronLeft size={18} />
          </button>
          <button
            className="account-menu"
            onClick={() => {
              setModal('');
              navigate('/dashboard');
            }}
          >
            <LayoutGrid size={19} />
            جلسه‌های من
            <ChevronLeft size={18} />
          </button>
          <button className="account-menu danger" onClick={() => void logout()}>
            <LogOut size={19} />
            خروج از حساب
          </button>
        </Modal>
      )}
      {modal === 'invite' && room && (
        <Modal title="یک صندلی برای هم‌تیمی‌ها" onClose={closeModal}>
          <div className="invite-illustration">
            <Avatar name="شما" color="violet" />
            <span>+</span>
            <Avatar name="تیم" color="mint" />
          </div>
          <p className="muted">
            {demo
              ? 'برای دعوت اعضای واقعی، ابتدا اتاق خودتان را بسازید.'
              : 'این لینک را برای تیمت بفرست. اعضا پس از ورود با شماره موبایل به اتاق می‌پیوندند.'}
          </p>
          {demo ? (
            <button className="primary full" onClick={needCreate}>
              ساخت اتاق واقعی
              <Plus size={17} />
            </button>
          ) : (
            <>
              <div className="invite-link">
                <Link2 size={18} />
                <input
                  aria-label="لینک دعوت"
                  dir="ltr"
                  readOnly
                  value={location.origin + `/room/${room.id}`}
                />
              </div>
              <button
                className="primary full"
                onClick={() => {
                  if (!navigator.clipboard) {
                    document
                      .querySelector<HTMLInputElement>('input[aria-label="لینک دعوت"]')
                      ?.select();
                    notify('لینک را از کادر بالا انتخاب و کپی کنید.');
                    return;
                  }
                  void navigator.clipboard
                    .writeText(location.origin + `/room/${room.id}`)
                    .then(() => notify('لینک دعوت کپی شد.'))
                    .catch(() => notify('لینک را از کادر بالا انتخاب و کپی کنید.'));
                }}
              >
                <Copy size={17} />
                کپی لینک دعوت
              </button>
            </>
          )}
        </Modal>
      )}
      {modal === 'room-theme' && room && (
        <Modal title="رنگی برای میزِ تیم شما" onClose={closeModal}>
          <p className="muted">تم انتخابی برای همهٔ اعضای اتاق به‌صورت زنده نمایش داده می‌شود.</p>
          <ThemePicker
            value={roomTheme(room.theme)}
            disabled={busy}
            readOnly={room.host_id !== shownUser?.id}
            onChange={(theme) =>
              void command({ action: 'set_theme', value: theme }).catch(() => {})
            }
          />
          <p className="theme-picker-note">حالت روشن و تیره فقط نمایش شما را تغییر می‌دهد.</p>
        </Modal>
      )}
      {modal === 'settings' && room && (
        <Modal title="تنظیمات جلسه" onClose={closeModal}>
          <VisibilityPicker
            value={room.visibility}
            disabled={busy || room.closed || room.host_id !== shownUser?.id || demo}
            onChange={(visibility) =>
              void command({ action: 'set_visibility', value: visibility }).catch(() => {})
            }
          />
          <p className="theme-picker-note">
            {room.visibility === 'public'
              ? 'این اتاق در صفحهٔ اصلی برای همه نمایش داده می‌شود.'
              : 'برای پیوستن به این اتاق، لینک دعوت لازم است.'}
          </p>
          <ThemePicker
            value={roomTheme(room.theme)}
            disabled={busy}
            readOnly={room.host_id !== shownUser?.id}
            onChange={(theme) =>
              void command({ action: 'set_theme', value: theme }).catch(() => {})
            }
          />
          {room.host_id !== shownUser?.id && (
            <p className="theme-picker-note">تغییر رنگ تم فقط برای سازندهٔ اتاق مجاز است.</p>
          )}
          <div className="setting-row">
            <span>دسته کارت</span>
            <b>{room.deck === 'fibonacci' ? 'فیبوناچی' : 'اندازه تی‌شرت'}</b>
          </div>
          <div className="setting-row">
            <span>نمایش کارت‌ها</span>
            <b>فقط گرداننده</b>
          </div>
          <div className="setting-row">
            <span>ظرفیت اتاق</span>
            <b>۲۰ نفر</b>
          </div>
          <div className="setting-row">
            <span>رأی‌گیری مستقل</span>
            <ShieldCheck size={19} />
          </div>
          {room.host_id === shownUser?.id && !demo && !room.closed && (
            <>
              <p className="muted">
                با پایان جلسه، رأی‌گیری بسته می‌شود و نتیجه‌ها برای مشاهده و خروجی باقی می‌مانند.
              </p>
              <button className="danger-button full" onClick={() => setModal('close-confirm')}>
                <Flag size={18} />
                پایان جلسه
              </button>
            </>
          )}
          {demo && <p className="muted">در اتاق نمونه فقط جریان تخمین را امتحان کنید.</p>}
        </Modal>
      )}
      {modal === 'close-confirm' && (
        <Modal title="پایان این جلسه؟" onClose={closeModal}>
          <p className="muted">
            پس از پایان، رأی جدید ثبت نمی‌شود. همه تخمین‌های ثبت‌شده در تاریخچه باقی می‌مانند.
          </p>
          <Button
            busy={busy}
            className="danger-button full"
            onClick={() =>
              void command({ action: 'close' })
                .then(() => {
                  setModal('');
                  notify('جلسه پایان یافت.');
                })
                .catch(() => {})
            }
          >
            بله، پایان جلسه
            <Flag size={18} />
          </Button>
        </Modal>
      )}
      {modal === 'help' && (
        <Modal title="هر کارت، یک دیدگاه" onClose={closeModal}>
          <div className="help-steps">
            {[
              [
                '۱',
                'یک تسک، یک گفت‌وگو',
                'گرداننده موضوع تخمین را اضافه می‌کند و تیم درباره محدوده آن صحبت می‌کند.',
              ],
              [
                '۲',
                'مستقل انتخاب کنید',
                'یک کارت را بر اساس پیچیدگی انتخاب کنید. با کلیک دوباره می‌توانید رأی را بردارید.',
              ],
              [
                '۳',
                'کارت‌ها را رو کنید',
                'گرداننده رأی‌ها را نمایش می‌دهد. اختلاف رأی‌ها را بررسی کنید و در صورت نیاز دوباره رأی بدهید.',
              ],
              [
                '۴',
                'به یک تصمیم برسید',
                'گرداننده تخمین نهایی را ثبت می‌کند و تسک بعدی را شروع می‌کند.',
              ],
            ].map(([n, title, desc]) => (
              <div key={n}>
                <span>{n}</span>
                <section>
                  <h3>{title}</h3>
                  <p>{desc}</p>
                </section>
              </div>
            ))}
          </div>
          <p className="help-special">
            <b>?</b> نیاز به اطلاعات بیشتر <span>·</span>
            <Coffee size={17} /> یک استراحت کوتاه
          </p>
        </Modal>
      )}
      {toast && (
        <div className="toast" role="status">
          <CheckCircle2 size={18} />
          {toast}
          <button aria-label="بستن پیام" onClick={() => setToast('')}>
            <X size={16} />
          </button>
        </div>
      )}
    </div>
  );
}
