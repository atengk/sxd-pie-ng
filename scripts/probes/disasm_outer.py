import struct
import capstone

with open(r'C:\software\疯玩神仙道\pie.exe', 'rb') as f:
    # 1. Read cnajrruv (0x84D000)
    f.seek(0x00302800)
    entry_code = f.read(0x200)

    # 2. Read qvngnnjm (0x6CE000)
    f.seek(0x00184600)
    qv_data = bytearray(f.read(0x1000))

# Decrypt first 0x1000 bytes of qvngnnjm
key1 = 0x5e25b697
key2 = 0x6bbda236
for i in range(0, len(qv_data), 4):
    val = struct.unpack('<I', qv_data[i:i+4])[0]
    val ^= key1
    val = (val + key2) & 0xFFFFFFFF
    qv_data[i:i+4] = struct.pack('<I', val)

md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)

print("=== Disassembly of Entry Point (0x0084D000) ===")
for insn in md.disasm(entry_code[:0x75], 0x0084D000):
    print(f"0x{insn.address:08X}:  {insn.mnemonic:<8} {insn.op_str}")

print("\n=== Disassembly of Decrypted Stage 1 (0x006CE000) ===")
for insn in md.disasm(bytes(qv_data[:0x120]), 0x006CE000):
    print(f"0x{insn.address:08X}:  {insn.mnemonic:<8} {insn.op_str}")
