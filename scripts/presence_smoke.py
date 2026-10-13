#!/usr/bin/env python3
"""Presence lifecycle against an explicitly selected local development / QA server."""
import http.client
import http.cookiejar
import json
import secrets
import socket
import sys
import threading
import time
import urllib.error
import urllib.parse
import urllib.request

base = sys.argv[1].rstrip('/')
url = urllib.parse.urlsplit(base)
assert url.hostname in ('127.0.0.1', 'localhost'), 'Local QA server required'
assert b'development' in urllib.request.urlopen(base + '/api/health', timeout=5).read()


def request(client, path, method='GET', body=None, status=200):
    req = urllib.request.Request(base + '/api' + path, method=method,
        data=None if method == 'GET' else json.dumps(body or {}).encode(),
        headers={'Content-Type': 'application/json'})
    try:
        response = client.open(req, timeout=8)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        data = json.loads(response.read() or 'null')
        assert response.status == status, (path, response.status, data)
        return data


def login(name):
    jar = http.cookiejar.CookieJar()
    client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    phone = '0919' + ''.join(str(secrets.randbelow(10)) for _ in range(7))
    challenge = request(client, '/auth/otp', 'POST', {'phone': phone})
    user = request(client, '/auth/verify', 'POST', {'phone': phone, 'code': challenge['dev_code']})
    user = request(client, '/me', 'PATCH', {'name': name, 'avatar': 'violet', 'role': 'QA'})
    return client, user, '; '.join(c.name + '=' + c.value for c in jar)


host, hu, hc = login('Presence QA host')
guest, gu, gc = login('Presence QA guest')
room = request(host, '/rooms', 'POST', {'name': 'Presence lifecycle QA', 'deck': 'fibonacci', 'theme': 'violet'}, status=201)
path = '/rooms/' + room['id']
request(guest, path + '/join', 'POST', {'observer': False})
room = request(host, path + '/commands', 'POST', {'action': 'add_round', 'title': 'Live electorate'})
vote = {'action': 'vote', 'round_id': room['active_round_id'], 'value': '8'}
assert request(guest, path + '/commands', 'POST', vote, 409)['error']['code'] == 'disconnected'
streams = []


class Stream:
    def __init__(self, cookie):
        self.connection = http.client.HTTPConnection(url.hostname, url.port, timeout=20)
        self.connection.request('GET', '/api' + path + '/events', headers={'Cookie': cookie})
        self.socket = self.connection.sock
        self.response = self.connection.getresponse()
        assert self.response.status == 200
        self.ready = threading.Event()
        self.thread = threading.Thread(target=self.read, daemon=True)
        self.thread.start()
        streams.append(self)
        assert self.ready.wait(5), 'No initial room snapshot'

    def read(self):
        try:
            while line := self.response.readline():
                if line.startswith(b'data: '):
                    self.ready.set()
        except (OSError, ValueError):
            pass

    def close(self):
        if self.socket:
            try:
                self.socket.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
        self.thread.join(2)
        self.response.close()
        self.connection.close()


def wait_online(expected):
    deadline = time.monotonic() + 35
    while time.monotonic() < deadline:
        snapshot = request(host, path)
        found = next(m for m in snapshot['members'] if m['id'] == gu['id'])['online']
        if found == expected:
            return snapshot
        time.sleep(.1)
    raise AssertionError('Presence did not change')


try:
    hs = Stream(hc)
    first, last = Stream(gc), Stream(gc)
    request(guest, path + '/commands', 'POST', vote)
    first.close()
    assert wait_online(True)['my_vote'] == ''
    last.close()
    wait_online(False)
    assert request(guest, path + '/commands', 'POST', vote, 409)['error']['code'] == 'disconnected'
    assert request(host, path + '/commands', 'POST', {'action': 'reveal'}, 409)['error']['code'] == 'no_votes'
    request(host, path + '/commands', 'POST', {**vote, 'value': '5'})
    revealed = request(host, path + '/commands', 'POST', {'action': 'reveal'})
    assert revealed['distribution'] == {'5': 1} and revealed['average'] == 5
    back = Stream(gc)
    wait_online(True)
    assert request(guest, path)['my_vote'] == '8'
    assert request(host, path)['distribution'] == {'5': 1, '8': 1}
    request(host, path + '/commands', 'POST', {'action': 'estimate', 'value': '5'})
    back.close()
    wait_online(False)
    assert request(host, path)['distribution'] == {'5': 1, '8': 1}
    request(host, path + '/commands', 'POST', {'action': 'close'})
    print('PASS: offline vote blocked; two tabs retain one seat; final tab releases presence')
    print('PASS: offline votes excluded; reconnection restores membership/vote; completed history preserved')
finally:
    for stream in streams:
        stream.close()
