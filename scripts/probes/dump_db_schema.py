import sqlite3
import os
import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

db_path = r"C:\software\疯玩神仙道\Pieb.db"
conn = sqlite3.connect(db_path)
cur = conn.cursor()

# 1. Get all tables and their row counts
cur.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")
tables = [r[0] for r in cur.fetchall()]

table_info = {}
for t in tables:
    cur.execute(f"SELECT COUNT(*) FROM `{t}`;")
    count = cur.fetchone()[0]
    
    cur.execute(f"PRAGMA table_info(`{t}`);")
    cols = [{"cid": c[0], "name": c[1], "type": c[2], "notnull": c[3], "pk": c[5]} for c in cur.fetchall()]
    
    # Sample 2 rows
    cur.execute(f"SELECT * FROM `{t}` LIMIT 2;")
    sample_rows = cur.fetchall()
    
    table_info[t] = {
        "count": count,
        "columns": [c["name"] + f" ({c['type']})" for c in cols],
        "sample": sample_rows
    }

print(f"=== Total Tables: {len(tables)} ===")
for t, info in table_info.items():
    print(f"\n[Table: {t}] - Rows: {info['count']}")
    print(f"  Columns: {', '.join(info['columns'][:10])}{' ...' if len(info['columns']) > 10 else ''}")
    if info['sample']:
        print(f"  Sample row 1: {info['sample'][0]}")
