import os
import glob
import re
import sys

sys.stdout.reconfigure(encoding='utf-8')

log_files = glob.glob(r"C:\software\疯玩神仙道\*.log") + glob.glob(r"C:\software\疯玩神仙道\temp\*.log")
print("Found log files:", len(log_files))

net_lines = []
for lf in log_files:
    try:
        with open(lf, 'r', encoding='gbk', errors='replace') as f:
            for line in f:
                l = line.strip()
                if any(k in l for k in ['协议', '连接', '登录', '断开', 'socket', 'http', 'IP', '端口', '封包', '握手', '重连', 'token', 'code', '发送', '接收', '完成']):
                    net_lines.append(l)
    except Exception as e:
        print(f"Error reading {lf}: {e}")

print(f"Total matching network/protocol lines: {len(net_lines)}")
# Sample distinctive protocol lines
seen = set()
distinct = []
for l in net_lines:
    # remove timestamp
    content = re.sub(r'\[\d{2}:\d{2}:\d{2}\]\s*', '', l)
    if content not in seen:
        seen.add(content)
        distinct.append(content)

print("\nDistinct Protocol / State log entries:")
for d in distinct[:40]:
    print(" -", d)
if len(distinct) > 40:
    print(f"... ({len(distinct) - 40} more entries)")
