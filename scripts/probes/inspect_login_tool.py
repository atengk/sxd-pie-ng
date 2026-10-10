import os
import re
import sys

sys.stdout.reconfigure(encoding='utf-8')

login_tool_path = r"C:\software\疯玩神仙道\Sxd Assistant Login Tool.exe"

with open(login_tool_path, 'rb') as f:
    data = f.read()

# 1. Search for URLs and web endpoints
urls = re.findall(rb'https?://[^\s\x00\"\'<>]+', data)
print(f"=== Found {len(urls)} URLs in Login Tool ===")
for u in set(urls):
    try:
        print("  URL:", u.decode('utf-8'))
    except:
        print("  URL (raw):", u)

# 2. Search for HTTP headers / requests (POST, GET, User-Agent, Cookie)
http_strings = re.findall(rb'(?:POST|GET|Cookie|User-Agent|Content-Type|Host:|HTTP/1\.[01])[^\x00\r\n]{0,100}', data, re.IGNORECASE)
print(f"\n=== Found {len(http_strings)} HTTP-related strings ===")
for s in set(http_strings):
    try:
        print("  HTTP:", s.decode('latin1'))
    except:
        pass

# 3. Search for interesting keywords: token, hash, login, server, gate, md5, key, password
keywords = [b'login', b'server', b'gate', b'user.ini', b'account.ini', b'pie.exe', b'3fangyuan', b'fengwan', b'sign', b'token']
print("\n=== Keyword occurrences ===")
for kw in keywords:
    matches = [m.start() for m in re.finditer(re.escape(kw), data, re.IGNORECASE)]
    print(f"  Keyword '{kw.decode()}': {len(matches)} occurrences")
    for pos in matches[:3]:
        # print surrounding text
        chunk = data[max(0, pos-20): min(len(data), pos+60)]
        printable = ''.join(chr(b) if 32 <= b <= 126 else '.' for b in chunk)
        print(f"    @ 0x{pos:X}: {printable}")
