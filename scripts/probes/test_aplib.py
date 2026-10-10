import struct

def decompress_aplib(src):
    src_idx = 0
    tag = 0
    bit_count = 0
    dst = bytearray()
    ebx = 2 # last operation state
    ebp = 0 # last offset

    def get_bit():
        nonlocal src_idx, tag, bit_count
        if bit_count == 0:
            tag = src[src_idx]
            src_idx += 1
            bit_count = 8
        bit = (tag >> 7) & 1
        tag = (tag << 1) & 0xFF
        bit_count -= 1
        return bit

    def get_gamma():
        gamma = 1
        while True:
            gamma = (gamma << 1) | get_bit()
            if get_bit() == 0:
                break
        return gamma

    # The exact assembly at 0x6ce0bf:
    # Let's emulate the exact logic of the asm
    class AsmDecompressor:
        def __init__(self, data):
            self.src = data
            self.esi = 0
            self.dst = bytearray()
            self.dl = 0x80
            self.ebx = 2
            self.ebp = 0

        def next_tag_bit(self):
            # add dl, dl
            cf = (self.dl & 0x80) != 0
            self.dl = (self.dl << 1) & 0xFF
            if self.dl == 0:
                # mov dl, [esi]; inc esi; adc dl, dl
                new_byte = self.src[self.esi]
                self.esi += 1
                cf = (new_byte & 0x80) != 0
                self.dl = ((new_byte << 1) | 1) & 0xFF
            return cf

        def run(self, max_out=0x300000):
            # 0x6ce0cb: literal
            literal = True
            while len(self.dst) < max_out and self.esi < len(self.src):
                if literal:
                    al = self.src[self.esi]
                    self.esi += 1
                    self.dst.append(al)
                    self.ebx = 2
                    literal = False

                # 0x6ce0d6: nexttag
                bit1 = self.next_tag_bit()
                if not bit1: # jae literal (CF=0)
                    literal = True
                    continue

                # 0x6ce0e1:
                bit2 = self.next_tag_bit()
                if not bit2: # jae 0x6ce13b (CF=0)
                    # 0x6ce13b:
                    eax = 1
                    while True:
                        b = self.next_tag_bit()
                        eax = (eax << 1) | (1 if b else 0)
                        b = self.next_tag_bit()
                        if not b: # jb loop (CF=1)
                            break
                    eax -= self.ebx
                    self.ebx = 1
                    if eax == 0:
                        # 0x6ce15f: match with length using previous offset (ebp)
                        ecx = 1
                        while True:
                            b = self.next_tag_bit()
                            ecx = (ecx << 1) | (1 if b else 0)
                            b = self.next_tag_bit()
                            if not b:
                                break
                        # copy ecx bytes from dst - ebp
                        match_off = self.ebp
                        for _ in range(ecx):
                            self.dst.append(self.dst[-match_off])
                        self.ebx = 2
                        continue
                    else:
                        # 0x6ce187:
                        eax -= 1
                        eax = (eax << 8) | self.src[self.esi]
                        self.esi += 1
                        self.ebp = eax # new offset
                        ecx = 1
                        while True:
                            b = self.next_tag_bit()
                            ecx = (ecx << 1) | (1 if b else 0)
                            b = self.next_tag_bit()
                            if not b:
                                break
                        if eax >= 0x7D00:
                            ecx += 2
                        elif eax >= 0x500:
                            ecx += 1
                        elif eax <= 0x7F:
                            ecx += 2
                        
                        match_off = self.ebp
                        for _ in range(ecx):
                            self.dst.append(self.dst[-match_off])
                        self.ebx = 2
                        continue

                # 0x6ce0ec:
                eax = 0
                bit3 = self.next_tag_bit()
                if not bit3:
                    # 0x6ce1dc:
                    al = self.src[self.esi]
                    self.esi += 1
                    match_off = al >> 1
                    if match_off == 0:
                        # end of stream! (je 0x6ce1fd)
                        print(f"End of stream tag reached at src offset {self.esi}, decompressed {len(self.dst)} bytes")
                        break
                    ecx = 2 + (al & 1)
                    self.ebp = match_off
                    for _ in range(ecx):
                        self.dst.append(self.dst[-match_off])
                    self.ebx = 1
                    continue
                else:
                    # read 4 bits:
                    for _ in range(4):
                        b = self.next_tag_bit()
                        eax = (eax << 1) | (1 if b else 0)
                    if eax == 0:
                        self.dst.append(0)
                    else:
                        self.dst.append(self.dst[-eax])
                    self.ebx = 2
                    continue

    dec = AsmDecompressor(src)
    dec.run()
    return bytes(dec.dst)

with open(r'C:\software\疯玩神仙道\pie.exe', 'rb') as f:
    f.seek(0x00184600)
    qv_full = bytearray(f.read(0x17E200))

# Decrypt first 0x1000 bytes
key1 = 0x5e25b697
key2 = 0x6bbda236
for i in range(0, 0x1000, 4):
    val = struct.unpack('<I', qv_full[i:i+4])[0]
    val ^= key1
    val = (val + key2) & 0xFFFFFFFF
    qv_full[i:i+4] = struct.pack('<I', val)

comp_data = qv_full[0x212:]
print("Starting decompression...")
out = decompress_aplib(comp_data)
print(f"Decompressed output length: {len(out)} bytes")
if len(out) > 0:
    print("First 64 bytes of decompressed:", out[:64].hex(' '))
    # Search for strings in decompressed
    import re
    strs = re.findall(b'[\x20-\x7e]{5,}', out)
    print(f"Total ASCII strings found: {len(strs)}")
    for s in strs[:30]:
        print(" -", s.decode('latin1'))
