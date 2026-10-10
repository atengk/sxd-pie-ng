import re
import sys

sys.stdout.reconfigure(encoding='utf-8')

# Search for common SXD protocol strings in sxd_wd_V3.0_1680231221.exe
wd_path = r"C:\software\疯玩神仙道\temp\sxd_wd_V3.0_1680231221.exe"

with open(wd_path, 'rb') as f:
    data = f.read()

# Search for Mod_ strings or Protocol names
mod_matches = re.findall(rb'Mod_[A-Za-z0-9_]+', data)
print(f"Mod_ matches in sxd_wd: {len(mod_matches)}")
if mod_matches:
    distinct_mods = sorted(list(set(m.decode('latin1') for m in mod_matches)))
    print("Found Module Identifiers:")
    for m in distinct_mods[:30]:
        print("  *", m)

# Search for Action / Protocol command strings
proto_words = [b'enter_town', b'sweep_mission', b'hunt_fate', b'get_player_info', b'chat_server', b'gateway', b'crossdomain']
print("\nProtocol keyword search in micro client:")
for w in proto_words:
    cnt = len(re.findall(re.escape(w), data, re.IGNORECASE))
    print(f"  {w.decode()}: {cnt} matches")
