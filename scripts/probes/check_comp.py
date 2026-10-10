import struct

with open(r'C:\software\疯玩神仙道\pie.exe', 'rb') as f:
    f.seek(0x00184600)
    qv_full = bytearray(f.read(0x17E200)) # full size of qvngnnjm

# Decrypt first 0x1000 bytes
key1 = 0x5e25b697
key2 = 0x6bbda236
for i in range(0, 0x1000, 4):
    val = struct.unpack('<I', qv_full[i:i+4])[0]
    val ^= key1
    val = (val + key2) & 0xFFFFFFFF
    qv_full[i:i+4] = struct.pack('<I', val)

# Look at 0x6CE212 (offset 0x212)
comp_start = 0x212
comp_bytes = qv_full[comp_start:comp_start + 64]
print("Compressed data start at 0x212 (hex):", comp_bytes.hex(' '))

# Let's write an exact aPLib decompressor in Python matching the assembly routine at 0x6ce0bf!
def decompress_aplib_asm(src):
    class BitStream:
        def __init__(self, data):
            self.data = data
            self.src_ptr = 0
            self.tag = 0x80

        def get_bit(self):
            # add dl, dl
            self.tag = (self.tag << 1) & 0xFF
            carry = (self.tag > 0xFF) or bool(self.tag & 0x100) # wait, tag is 8-bit
            # in 8086: add dl, dl sets CF if bit 7 was 1, and ZF if result is 0
            # let's be very precise:
            # tag is 8-bit unsigned
            pass

    # Let's inspect the exact asm:
    # 0x6ce0c9: mov dl, 0x80
    # 0x6ce0cb: mov al, [esi]; inc esi; mov [edi], al; inc edi (literal)
    # 0x6ce0d1: mov ebx, 2
    # 0x6ce0d6: add dl, dl; jne 0x6ce0df; mov dl, [esi]; inc esi; adc dl, dl
    # 0x6ce0df: jae 0x6ce0cb (if CF=0, literal!)
    pass

print(f"Total bytes in qvngnnjm: {len(qv_full)}")
