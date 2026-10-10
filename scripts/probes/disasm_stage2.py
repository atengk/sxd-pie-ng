import struct
import capstone

with open(r'C:\software\疯玩神仙道\pie.exe', 'rb') as f:
    f.seek(0x00184600)
    qv_data = bytearray(f.read(0x1000))

key1 = 0x5e25b697
key2 = 0x6bbda236
for i in range(0, len(qv_data), 4):
    val = struct.unpack('<I', qv_data[i:i+4])[0]
    val ^= key1
    val = (val + key2) & 0xFFFFFFFF
    qv_data[i:i+4] = struct.pack('<I', val)

md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)

print("=== Disassembly of Decrypted Stage 1 from 0x006CE05E ===")
for insn in md.disasm(bytes(qv_data[0x5E:0x200]), 0x006CE05E):
    print(f"0x{insn.address:08X}:  {insn.mnemonic:<8} {insn.op_str}")
