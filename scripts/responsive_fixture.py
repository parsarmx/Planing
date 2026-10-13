#!/usr/bin/env python3
"""Create isolated, synthetic responsive QA data; never edits app development data."""
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path

path = Path(sys.argv[1] if len(sys.argv) > 1 else '/tmp/poker-responsive-state.json').resolve()
if path.exists():
    raise SystemExit('Choose a new fixture path; existing files are never overwritten.')
now = datetime.now(timezone.utc).isoformat()
room_id = '00000000000000000000000000000001'
users = {}
members = []
colors = ['violet', 'pink', 'blue', 'mint', 'peach', 'yellow']
for i in range(20):
    uid = f'qa-user-{i:02d}'
    users[uid] = {'id': uid, 'phone': f'+9891200088{i:02d}',
                 'name': ('کاربرآزمایشیبانامنمایشیبسیارطولانی' if i else 'کاربر تست ریسپانسیو') + f' {i+1}',
                 'avatar': colors[i % len(colors)], 'role': 'همکار تیم محصول با عنوان شغلی طولانی برای بررسی موبایل', 'created_at': now}
    members.append({'user_id': uid, 'observer': i == 19})
rounds = []
for i in range(24):
    rid = f'qa-round-{i:02d}'
    rounds.append({'id': rid,
                   'title': ('تسک‌خیلی‌طولانی‌بدون‌فاصله‌برای‌بررسی‌نمایش‌در‌موبایل' if i == 1 else
                             'بررسی ورود، پروفایل، تسک‌ها و هماهنگی اعضای تیم در گوشی‌های کوچک و بزرگ') + f' · {i+1}',
                   'description': ('توضیحات طولانی تسک برای بررسی خوانایی متن و دسترسی به کنترل‌ها. ' * 30) + '\nhttps://example.test/' + 'long-path-' * 30,
                   'votes': {f'qa-user-{n:02d}': ['0', '1', '2', '3', '5', '8', '13', '21', '34', '?', '☕'][n % 11] for n in range(19)} if i == 0 else {},
                   'status': 'voting' if i == 0 else 'pending', 'estimate': '', 'started_at': now})
room = {'id': room_id, 'name': 'جلسه‌برنامه‌ریزی‌بانام‌طولانی‌برای‌آزمایش‌ریسپانسیو‌در‌گوشی‌های‌متفاوت',
        'host_id': 'qa-user-00', 'deck': 'fibonacci', 'theme': 'pink', 'members': members,
        'rounds': rounds, 'active_round_id': 'qa-round-00', 'created_at': now, 'closed': False, 'revision': 1}
state = {'users': users, 'otps': {}, 'sessions': {}, 'rooms': {room_id: room}}
path.parent.mkdir(parents=True, exist_ok=True)
with path.open('x') as f:
    os.chmod(path, 0o600)
    json.dump(state, f, ensure_ascii=False)
print('Fixture:', path)
print('Login phone: 09120008800 (development OTP)')
print('Room:', room_id)
print('Synthetic members: 20; long tasks: 24')
