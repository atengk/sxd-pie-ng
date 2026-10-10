import struct

with open(r'C:\software\疯玩神仙道\pie.exe', 'rb') as f:
    f.seek(0x00184600) # RawPtr of section qvngnnjm (VA 0x6CE000)
    data = bytearray(f.read(0x1000))

# Decryption routine:
# ecx = size >> 2
# eax = 0x5e25b697
# ebx = 0x6bbda236
# for each dword:
#   val ^= eax
#   val = (val + ebx) & 0xFFFFFFFF
key1 = 0x5e25b697
key2 = 0x6bbda236

for i in range(0, len(data), 4):
    val = struct.unpack('<I', data[i:i+4])[0]
    val ^= key1
    val = (val + key2) & 0xFFFFFFFF
    data[i:i+4] = struct.pack('<I', val)

print("Decrypted first 256 bytes (hex):")
for i in range(0, 256, 16):
    chunk = data[i:i+16]
    hex_str = ' '.join(f'{b:02x}' for b in chunk)
    asc_str = ''.join(chr(b) if 32 <= b <= 126 else '.' for b in chunk)
    print(f'{0x006CE000 + i:08x}:  {hex_str:<48}  {asc_str}')

# Search for strings in decrypted 4KB
import re
strs = re.findall(b'[\x20-\x7e]{4,}', data)
print("\nStrings found in decrypted chunk:")
for s in strs:
    print(" -", s.decode('ascii'))
