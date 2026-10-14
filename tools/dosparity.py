"""dosparity.py GAMEDIR OUTDIR STEP...: run the original game in DOSBox-X (headless, no host
input, QMP on 127.0.0.1:4444), apply steps and save screen dumps for parity checks.

Steps: wait:SECONDS  key:NAME (QMP key name, pressed ~150 ms)  text:STRING  shot:NAME
The game runs from a temporary copy of GAMEDIR, started as START.EXE STING (no title)."""
import os, shutil, subprocess, sys, tempfile, time
sys.path.insert(0, os.path.expanduser('~/tools/dos-mcp/src'))
from dos_mcp.qmp import QmpClient
from dos_mcp import tools

DOSBOX = os.path.expanduser('~/tools/dosbox-x-remotedebug/src/dosbox-x')


# EGA palette (the 64-colour rgbRGB values of the attribute palette entries).
def ega_rgb(v):
    r = 0x55 * (((v >> 2) & 1) * 2 + ((v >> 5) & 1))
    g = 0x55 * (((v >> 1) & 1) * 2 + ((v >> 4) & 1))
    b = 0x55 * ((v & 1) * 2 + ((v >> 3) & 1))
    return r, g, b


def save_ega(dumpfile, base):
    """Decode a 320x200 16-colour EGA screen (mode 0Dh) from a vgadump to base.png and base.raw."""
    data = open(dumpfile, 'rb').read()
    nl = data.index(b'\n')
    hdr = dict(kv.split('=') for kv in data[:nl].decode().split())
    mem = data[nl + 1:]
    start, offset = int(hdr['start']), int(hdr['offset'])
    pal = [int(x) for x in hdr['pal'].split(',')]
    stride = offset * 2 if offset else 40
    idx = bytearray(320 * 200)
    for y in range(200):
        for xb in range(40):
            a = (start + y * stride + xb) * 4
            p0, p1, p2, p3 = mem[a], mem[a + 1], mem[a + 2], mem[a + 3]
            for bit in range(8):
                m = 0x80 >> bit
                c = (p0 & m and 1) | (p1 & m and 2) | (p2 & m and 4) | (p3 & m and 8)
                idx[y * 320 + xb * 8 + bit] = c
    open(base + '.raw', 'wb').write(bytes(idx))
    try:
        from PIL import Image
        im = Image.new('RGB', (320, 200))
        im.putdata([ega_rgb(pal[c]) for c in idx])
        im.save(base + '.png')
    except ImportError:
        pass


def main():
    game, out, steps = sys.argv[1], sys.argv[2], sys.argv[3:]
    os.makedirs(out, exist_ok=True)
    tmp = tempfile.mkdtemp(prefix='cotab-dos-')
    for f in os.listdir(game):
        p = os.path.join(game, f)
        if os.path.isfile(p):
            shutil.copy(p, tmp)
            os.chmod(os.path.join(tmp, f), 0o644)  # TP opens data files read/write
    conf = os.path.join(tmp, 'dosbox.conf')
    with open(conf, 'w') as f:
        f.write('[sdl]\noutput=surface\n[dosbox]\nmachine=svga_s3\nmemsize=16\nqmpserver=true\nqmpport=4444\n'
                '[cpu]\ncore=normal\ncycles=max\n[dos]\nxms=true\nems=true\n[autoexec]\nmount c "%s"\nc:\n%s\n' % (tmp, os.environ.get('DOSPAR_CMD', 'START.EXE STING')))
    env = dict(os.environ, SDL_VIDEODRIVER=os.environ.get('DOSPAR_VIDEO', 'dummy'), DOSBOX_NO_HOST_INPUT='1')
    proc = subprocess.Popen([DOSBOX, '-conf', conf], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        client = QmpClient(port=4444, timeout=35)
        for _ in range(100):
            try:
                tools.status(client)
                break
            except Exception:
                time.sleep(0.2)
        for s in steps:
            kind, _, arg = s.partition(':')
            if kind == 'wait':
                time.sleep(float(arg))
            elif kind == 'key':
                tools.key_event(client, arg, down=True)
                time.sleep(0.15)
                tools.key_event(client, arg, down=False)
                time.sleep(0.15)
            elif kind == 'text':
                for ch in arg:
                    k = ch.lower() if ch != ' ' else 'spc'
                    tools.key_event(client, k, down=True)
                    time.sleep(0.12)
                    tools.key_event(client, k, down=False)
                    time.sleep(0.12)
            elif kind == 'text80':
                tools.pause(client)
                r = tools.read_memory(client, 0xB8000, 4000)
                tools.resume(client)
                b = bytes.fromhex(r['data_hex'])
                for row in range(25):
                    line = ''.join(chr(c) if 32 <= c < 127 else ' ' for c in b[row * 160:row * 160 + 160:2]).rstrip()
                    if line:
                        print(line)
            elif kind == 'peek':
                a, n = arg.split(',')
                r = tools.read_memory(client, int(a, 16), int(n), allow_running=True)
                print('peek', a, r['data_hex'])
            elif kind == 'status':
                st = tools.status(client)
                print(st)
            elif kind == 'stack':
                st = tools.status(client)['cpu']
                a = st['ss'] * 16 + st['sp']
                r = tools.read_memory(client, a, 64, allow_running=True)
                b = bytes.fromhex(r['data_hex'])
                print('cs:ip %04X:%04X' % (st['cs'], st['ip']), ' '.join('%04X' % (b[i] | b[i + 1] << 8) for i in range(0, 64, 2)))
            elif kind == 'mode':
                r = tools.read_memory(client, 0x449, 1, allow_running=True)
                print('video mode', r['data_hex'])
            elif kind == 'vga':
                dumpfile = os.path.join(tmp, 'vga.bin')
                with client.session() as session:
                    session.execute('vgadump', {'file': dumpfile})
                shutil.copy(dumpfile, os.path.join(out, arg + '.bin'))
                save_ega(dumpfile, os.path.join(out, arg))
                print('vga', arg)
            elif kind == 'shot':
                open(os.path.join(out, arg + '.png'), 'wb').write(tools.capture_screen(client))
                print('shot', arg)
    finally:
        proc.terminate()
        proc.wait()
        shutil.rmtree(tmp, ignore_errors=True)


if __name__ == '__main__':
    main()
