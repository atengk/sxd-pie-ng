import os
import sys

sys.stdout.reconfigure(encoding='utf-8')

dir_path = r"C:\software\疯玩神仙道"

def read_file_preview(filename, max_lines=50, encoding='gbk'):
    p = os.path.join(dir_path, filename)
    if not os.path.exists(p):
        print(f"[-] {filename} does not exist.")
        return
    print(f"\n==================== [File: {filename}] (Size: {os.path.getsize(p)} bytes) ====================")
    try:
        with open(p, 'r', encoding=encoding, errors='replace') as f:
            lines = f.readlines()
            print(f"Total lines: {len(lines)}")
            for l in lines[:max_lines]:
                print(l.rstrip())
            if len(lines) > max_lines:
                print(f"... ({len(lines) - max_lines} more lines)")
    except Exception as e:
        print(f"Error reading {filename}: {e}")

# Read configs
for fn in ['pieb.ini', '01.ini', 'scra.ini', 'answers.txt', 'monkeyanswer.ini', 'user.ini']:
    read_file_preview(fn, max_lines=40)
