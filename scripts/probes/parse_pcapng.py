import struct
import os
import sys

sys.stdout.reconfigure(encoding='utf-8')

pcap_path = r"D:\sources\my\sxd-pie-ng\sxd.pcapng"
print(f"Pcap file size: {os.path.getsize(pcap_path)} bytes")

# Simple PCAPNG parser
# Enhanced Packet Block (EPB) block type: 0x00000006
with open(pcap_path, 'rb') as f:
    data = f.read()

pos = 0
packet_payloads = []
epb_count = 0

while pos < len(data) - 8:
    btype, blen = struct.unpack('<II', data[pos:pos+8])
    if blen == 0 or pos + blen > len(data):
        break
    
    if btype == 6: # EPB
        epb_count += 1
        # EPB format:
        # 0: type (4)
        # 4: block total length (4)
        # 8: interface id (4)
        # 12: timestamp high (4)
        # 16: timestamp low (4)
        # 20: captured packet length (4)
        # 24: original packet length (4)
        # 28: packet data (captured len)
        if blen >= 32:
            cap_len = struct.unpack('<I', data[pos+20:pos+24])[0]
            pkt_data = data[pos+28:pos+28+cap_len]
            # Try to parse IPv4 + TCP
            # Ethernet header: 14 bytes (0..14)
            if len(pkt_data) > 34:
                eth_type = struct.unpack('!H', pkt_data[12:14])[0]
                if eth_type == 0x0800: # IPv4
                    ihl = (pkt_data[14] & 0x0F) * 4
                    protocol = pkt_data[23]
                    src_ip = ".".join(str(b) for b in pkt_data[26:30])
                    dst_ip = ".".join(str(b) for b in pkt_data[30:34])
                    if protocol == 6: # TCP
                        tcp_start = 14 + ihl
                        if len(pkt_data) >= tcp_start + 20:
                            src_port = struct.unpack('!H', pkt_data[tcp_start:tcp_start+2])[0]
                            dst_port = struct.unpack('!H', pkt_data[tcp_start+2:tcp_start+4])[0]
                            data_offset = (pkt_data[tcp_start+12] >> 4) * 4
                            tcp_payload = pkt_data[tcp_start + data_offset:]
                            if len(tcp_payload) > 0:
                                packet_payloads.append({
                                    "src": f"{src_ip}:{src_port}",
                                    "dst": f"{dst_ip}:{dst_port}",
                                    "payload": tcp_payload
                                })
    pos += blen

print(f"Total EPB packets: {epb_count}, TCP payloads with data: {len(packet_payloads)}")

# Inspect the payloads
for i, p in enumerate(packet_payloads[:15]):
    plen = len(p["payload"])
    hex_head = p["payload"][:32].hex(' ')
    print(f"\n[Packet #{i+1}] {p['src']} -> {p['dst']} ({plen} bytes)")
    print(f"  Hex: {hex_head}")
    # Check if header matches ShenXianDao [Length (4B)] [Module (2B)] [Action (2B)]
    if plen >= 8:
        l, m, a = struct.unpack('>IHH', p["payload"][:8])
        print(f"  Decoded Header: Length={l}, Module={m} (0x{m:04X}), Action={a} (0x{a:04X})")
