import sys

sys.stdout.reconfigure(encoding='utf-8')

with open(r'C:\software\疯玩神仙道\pieb.ini', 'r', encoding='gbk', errors='replace') as f:
    lines = f.readlines()

current_sec = ""
sec_content = {}
for line in lines:
    l = line.strip()
    if l.startswith('[') and l.endswith(']'):
        current_sec = l[1:-1]
        sec_content[current_sec] = []
    elif current_sec and l and not l.startswith(';'):
        sec_content[current_sec].append(l)

for sec, items in sec_content.items():
    print(f"\n--- Section: [{sec}] ({len(items)} keys) ---")
    for item in items[:15]:
        print(" ", item)
    if len(items) > 15:
        print(f"  ... ({len(items) - 15} more)")
