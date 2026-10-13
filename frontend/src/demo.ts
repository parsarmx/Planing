import type { Room, Command } from './types';
import { roomTheme } from './themes';
import { consensusState } from './consensus';
export const demoUser = {
  id: 'demo-self',
  name: 'پارسا احمدی',
  phone: '',
  avatar: 'violet',
  role: 'توسعه‌دهنده',
};
export function demoRoom(): Room {
  return {
    id: 'demo',
    name: 'اسپرینت ۲۴ · تیم محصول',
    host_id: 'demo-self',
    deck: 'fibonacci',
    theme: 'violet',
    visibility: 'private',
    last_activity_at: new Date().toISOString(),
    cards: ['0', '1', '2', '3', '5', '8', '13', '21', '34', '?', '☕'],
    created_at: new Date().toISOString(),
    closed: false,
    revision: 1,
    my_vote: '',
    average: 0,
    consensus: false,
    distribution: {},
    active_round_id: 'd1',
    members: [
      { ...demoUser, observer: false, online: true, has_voted: false },
      {
        id: 'd-sara',
        name: 'سارا محمدی',
        avatar: 'peach',
        role: 'طراح محصول',
        observer: false,
        online: true,
        has_voted: true,
      },
      {
        id: 'd-ali',
        name: 'علی رضایی',
        avatar: 'blue',
        role: 'توسعه‌دهنده',
        observer: false,
        online: true,
        has_voted: true,
      },
      {
        id: 'd-nika',
        name: 'نیکا کریمی',
        avatar: 'pink',
        role: 'توسعه‌دهنده',
        observer: false,
        online: true,
        has_voted: false,
      },
      {
        id: 'd-amir',
        name: 'امیر حسینی',
        avatar: 'mint',
        role: 'توسعه‌دهنده',
        observer: false,
        online: true,
        has_voted: true,
      },
      {
        id: 'd-maryam',
        name: 'مریم موسوی',
        avatar: 'yellow',
        role: 'مدیر محصول',
        observer: true,
        online: true,
        has_voted: false,
      },
    ],
    rounds: [
      {
        id: 'd1',
        title: 'ورود و ثبت‌نام با شماره موبایل',
        description:
          'کاربر بتواند با شماره موبایل و کد یک‌بارمصرف وارد شود و پروفایل خود را تکمیل کند.',
        status: 'voting',
        estimate: '',
        started_at: new Date().toISOString(),
      },
      {
        id: 'd2',
        title: 'طراحی داشبورد شخصی کاربر',
        description: 'نمایش اتاق‌های فعال و جلسه‌های قبلی.',
        status: 'pending',
        estimate: '',
        started_at: '',
      },
      {
        id: 'd3',
        title: 'ارسال اعلان‌های درون‌برنامه‌ای',
        description: 'اعلان تغییر وضعیت و رویدادهای جدید.',
        status: 'pending',
        estimate: '',
        started_at: '',
      },
      {
        id: 'd4',
        title: 'بهبود تجربه پرداخت',
        description: 'بازطراحی مراحل پرداخت و وضعیت تراکنش‌ها.',
        status: 'completed',
        estimate: '3',
        started_at: '',
      },
    ],
  };
}
export function demoCommand(room: Room, c: Command): Room {
  const r = structuredClone(room);
  if (c.action === 'set_theme') {
    r.theme = roomTheme(c.value);
  }
  const active = r.rounds.find((x) => x.id === r.active_round_id)!;
  if (c.action === 'vote') {
    r.my_vote = c.value || '';
    r.members[0].has_voted = !!c.value;
  }
  if (c.action === 'reveal') {
    active.status = 'revealed';
    const simulatedVotes = r.my_vote === '5' ? ['', '5', '5', '5', '5'] : ['', '5', '8', '5', '5'];
    r.members.forEach((m, i) => {
      m.has_voted = !m.observer;
      m.vote = m.observer ? '' : i === 0 ? r.my_vote || '5' : simulatedVotes[i];
    });
    r.distribution = {};
    let sum = 0,
      n = 0;
    r.members.forEach((m) => {
      if (m.vote) {
        r.distribution[m.vote] = (r.distribution[m.vote] || 0) + 1;
        if (!isNaN(Number(m.vote))) {
          sum += Number(m.vote);
          n++;
        }
      }
    });
    r.average = n ? sum / n : 0;
  }
  if (c.action === 'reset') {
    active.status = 'voting';
    active.started_at = new Date().toISOString();
    r.my_vote = '';
    r.distribution = {};
    r.average = 0;
    r.members.forEach((m) => {
      m.has_voted = false;
      delete m.vote;
    });
  }
  if (c.action === 'estimate') {
    active.status = 'completed';
    active.estimate = c.value || '';
  }
  if (c.action === 'start_round') {
    r.active_round_id = c.round_id!;
    const next = r.rounds.find((x) => x.id === c.round_id)!;
    next.status = 'voting';
    next.started_at = new Date().toISOString();
    r.my_vote = '';
    r.members.forEach((m) => {
      m.has_voted = false;
      delete m.vote;
    });
    r.distribution = {};
    r.average = 0;
  }
  if (c.action === 'add_round') {
    const id = crypto.randomUUID();
    r.rounds.push({
      id,
      title: c.title!,
      description: c.description || '',
      status: 'pending',
      estimate: '',
      started_at: '',
    });
  }
  r.consensus = consensusState(r).value !== null;
  r.revision++;
  return r;
}
