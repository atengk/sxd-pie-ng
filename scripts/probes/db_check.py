import sqlite3
import os

db_path = r"C:\software\疯玩神仙道\Pieb.db"
if os.path.exists(db_path):
    conn = sqlite3.connect(db_path)
    cur = conn.cursor()
    cur.execute("SELECT name FROM sqlite_master WHERE type='table'")
    tables = cur.fetchall()
    print("Tables in Pieb.db:")
    for t in tables:
        print(" -", t[0])
else:
    print("Database not found.")
