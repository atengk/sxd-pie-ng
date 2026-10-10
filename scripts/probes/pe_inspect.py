import struct
import math
import sys
import os
import time

def calculate_entropy(data):
    if not data:
        return 0.0
    freq = {}
    for b in data:
        freq[b] = freq.get(b, 0) + 1
    entropy = 0.0
    length = len(data)
    for count in freq.values():
        p = count / length
        entropy -= p * math.log2(p)
    return entropy

def rva_to_offset(rva, sections):
    for sec in sections:
        va = sec['VirtualAddress']
        vs = sec['VirtualSize']
        raw_ptr = sec['PointerToRawData']
        raw_size = sec['SizeOfRawData']
        # Virtual size can be larger or smaller than raw size
        sec_size = max(vs, raw_size)
        if va <= rva < va + sec_size:
            offset = raw_ptr + (rva - va)
            return offset
    return None

def read_cstring(data, offset):
    end = data.find(b'\x00', offset)
    if end == -1:
        return ""
    try:
        return data[offset:end].decode('utf-8', errors='replace')
    except:
        return data[offset:end].decode('latin1', errors='replace')

def inspect_pe(file_path):
    with open(file_path, 'rb') as f:
        data = f.read()

    file_size = len(data)
    print(f"=== PE Architectural Reconnaissance for: {file_path} ===")
    print(f"File Size: {file_size} bytes ({file_size / (1024*1024):.2f} MB)")

    # 1. DOS Header
    if len(data) < 64 or data[:2] != b'MZ':
        print("[-] Error: Not a valid DOS MZ executable.")
        return

    e_lfanew = struct.unpack('<I', data[60:64])[0]
    print(f"[+] DOS Header valid, e_lfanew (PE Header Offset): 0x{e_lfanew:X} ({e_lfanew})")

    # Check for Rich Header in DOS Stub
    rich_offset = data.find(b'Rich', 0x40, e_lfanew)
    has_rich = False
    if rich_offset != -1:
        has_rich = True
        print(f"[+] Found Microsoft Rich Header near offset 0x{rich_offset:X} (MSVC Toolchain clue)")
    else:
        print("[-] No Microsoft Rich Header detected in DOS stub.")

    # 2. NT Headers
    if len(data) < e_lfanew + 4 or data[e_lfanew:e_lfanew+4] != b'PE\x00\x00':
        print("[-] Error: PE signature missing at e_lfanew.")
        return

    # File Header (20 bytes)
    file_header_offset = e_lfanew + 4
    machine, num_sections, timedatestamp, ptr_symbols, num_symbols, opt_header_size, characteristics = struct.unpack(
        '<HHIIIHH', data[file_header_offset:file_header_offset+20]
    )

    machine_str = "x86 (32-bit)" if machine == 0x014c else ("x64 (64-bit)" if machine == 0x8664 else f"Unknown (0x{machine:X})")
    ts_str = time.strftime('%Y-%m-%d %H:%M:%S UTC', time.gmtime(timedatestamp))

    print(f"\n--- COFF File Header ---")
    print(f"Target Machine       : 0x{machine:04X} ({machine_str})")
    print(f"Number of Sections   : {num_sections}")
    print(f"TimeDateStamp        : 0x{timedatestamp:08X} ({ts_str})")
    print(f"Size of Opt. Header  : {opt_header_size} bytes")
    print(f"Characteristics      : 0x{characteristics:04X}")
    char_flags = []
    if characteristics & 0x0001: char_flags.append("RELOCS_STRIPPED")
    if characteristics & 0x0002: char_flags.append("EXECUTABLE_IMAGE")
    if characteristics & 0x0004: char_flags.append("LINE_NUMS_STRIPPED")
    if characteristics & 0x0008: char_flags.append("LOCAL_SYMS_STRIPPED")
    if characteristics & 0x0020: char_flags.append("LARGE_ADDRESS_AWARE")
    if characteristics & 0x0100: char_flags.append("32BIT_MACHINE")
    if characteristics & 0x2000: char_flags.append("DLL")
    print(f"Characteristics Flags: {', '.join(char_flags)}")

    # 3. Optional Header
    opt_offset = file_header_offset + 20
    magic = struct.unpack('<H', data[opt_offset:opt_offset+2])[0]
    is_pe32_plus = (magic == 0x20B)
    opt_format = "PE32+ (64-bit)" if is_pe32_plus else ("PE32 (32-bit)" if magic == 0x10B else f"Unknown (0x{magic:X})")
    print(f"\n--- Optional Header ({opt_format}) ---")

    if not is_pe32_plus:
        major_linker, minor_linker = struct.unpack('<BB', data[opt_offset+2:opt_offset+4])
        size_code, size_init_data, size_uninit_data, entry_point, base_code, base_data = struct.unpack('<IIIIII', data[opt_offset+4:opt_offset+28])
        image_base, sec_align, file_align = struct.unpack('<III', data[opt_offset+28:opt_offset+40])
        os_major, os_minor, img_major, img_minor, sub_major, sub_minor = struct.unpack('<HHHHHH', data[opt_offset+40:opt_offset+52])
        win32_ver, size_image, size_headers, checksum = struct.unpack('<IIII', data[opt_offset+52:opt_offset+68])
        subsystem, dll_char = struct.unpack('<HH', data[opt_offset+68:opt_offset+72])
        dirs_offset = opt_offset + 96
        num_rva_sizes = struct.unpack('<I', data[opt_offset+92:opt_offset+96])[0]
    else:
        major_linker, minor_linker = struct.unpack('<BB', data[opt_offset+2:opt_offset+4])
        size_code, size_init_data, size_uninit_data, entry_point, base_code = struct.unpack('<IIIII', data[opt_offset+4:opt_offset+24])
        image_base, sec_align, file_align = struct.unpack('<QII', data[opt_offset+24:opt_offset+40])
        os_major, os_minor, img_major, img_minor, sub_major, sub_minor = struct.unpack('<HHHHHH', data[opt_offset+40:opt_offset+52])
        win32_ver, size_image, size_headers, checksum = struct.unpack('<IIII', data[opt_offset+52:opt_offset+68])
        subsystem, dll_char = struct.unpack('<HH', data[opt_offset+68:opt_offset+72])
        dirs_offset = opt_offset + 112
        num_rva_sizes = struct.unpack('<I', data[opt_offset+108:opt_offset+112])[0]

    subsystem_str = {1: "NATIVE", 2: "WINDOWS_GUI", 3: "WINDOWS_CUI (Console)", 7: "POSIX_CUI"}.get(subsystem, f"Other ({subsystem})")
    print(f"Linker Version       : {major_linker}.{minor_linker}")
    print(f"Address of EntryPoint: 0x{entry_point:08X} (RVA)")
    print(f"Image Base           : 0x{image_base:08X}")
    print(f"Section Alignment    : 0x{sec_align:X}")
    print(f"File Alignment       : 0x{file_align:X}")
    print(f"Size of Image        : 0x{size_image:X} ({size_image} bytes)")
    print(f"Size of Headers      : 0x{size_headers:X} ({size_headers} bytes)")
    print(f"Subsystem            : {subsystem_str}")
    print(f"DLL Characteristics  : 0x{dll_char:04X}")
    dll_flags = []
    if dll_char & 0x0040: dll_flags.append("DYNAMIC_BASE (ASLR)")
    if dll_char & 0x0080: dll_flags.append("FORCE_INTEGRITY")
    if dll_char & 0x0100: dll_flags.append("NX_COMPAT (DEP)")
    if dll_char & 0x0200: dll_flags.append("NO_ISOLATION")
    if dll_char & 0x0400: dll_flags.append("NO_SEH")
    if dll_char & 0x0800: dll_flags.append("NO_BIND")
    if dll_char & 0x2000: dll_flags.append("WDM_DRIVER")
    if dll_char & 0x8000: dll_flags.append("TERMINAL_SERVER_AWARE")
    print(f"Security Mitigations : {', '.join(dll_flags) if dll_flags else 'None (legacy)'}")

    # Data Directories
    dir_names = [
        "EXPORT", "IMPORT", "RESOURCE", "EXCEPTION", "CERTIFICATE", "BASE_RELOC",
        "DEBUG", "ARCHITECTURE", "GLOBAL_PTR", "TLS", "LOAD_CONFIG", "BOUND_IMPORT",
        "IAT", "DELAY_IMPORT", "CLR_RUNTIME_HEADER (.NET)", "RESERVED"
    ]
    data_dirs = {}
    print(f"\n--- Data Directories (Total: {num_rva_sizes}) ---")
    for i in range(min(num_rva_sizes, 16)):
        rva, sz = struct.unpack('<II', data[dirs_offset + i*8: dirs_offset + (i+1)*8])
        name = dir_names[i] if i < len(dir_names) else f"DIR_{i}"
        data_dirs[name] = {'rva': rva, 'size': sz}
        if rva != 0 or sz != 0:
            print(f"  [{i:02d}] {name:<26}: RVA 0x{rva:08X}, Size 0x{sz:08X} ({sz} bytes)")
        else:
            print(f"  [{i:02d}] {name:<26}: [Empty]")

    # Check CLR Runtime Header
    clr_dir = data_dirs.get("CLR_RUNTIME_HEADER (.NET)", {'rva': 0, 'size': 0})
    if clr_dir['rva'] == 0:
        print("\n[+] CLR Runtime Header is 0x0 => Confirmed: NOT a .NET assembly!")
    else:
        print(f"\n[!] CLR Runtime Header present (RVA 0x{clr_dir['rva']:08X}) => .NET / Managed code detected!")

    # 4. Section Headers
    sec_header_start = opt_offset + opt_header_size
    sections = []
    print(f"\n--- Section Table ({num_sections} sections) ---")
    print(f"{'Index':<5} {'Name':<10} {'VirtAddr':<12} {'VirtSize':<12} {'RawSize':<12} {'RawPtr':<12} {'Entropy':<8} {'Flags'}")
    print("-" * 90)

    for i in range(num_sections):
        offset = sec_header_start + i * 40
        raw_name = data[offset:offset+8].rstrip(b'\x00')
        sec_name = raw_name.decode('latin1', errors='replace')
        vsize, vaddr, raw_size, raw_ptr, reloc_ptr, line_ptr, num_reloc, num_line, flags = struct.unpack(
            '<IIIIIIHHI', data[offset+8:offset+40]
        )
        sec_bytes = data[raw_ptr:raw_ptr+raw_size] if raw_ptr + raw_size <= len(data) else b''
        ent = calculate_entropy(sec_bytes)

        flag_desc = []
        if flags & 0x00000020: flag_desc.append("CODE")
        if flags & 0x00000040: flag_desc.append("IDATA")
        if flags & 0x00000080: flag_desc.append("UDATA")
        if flags & 0x20000000: flag_desc.append("EXEC")
        if flags & 0x40000000: flag_desc.append("READ")
        if flags & 0x80000000: flag_desc.append("WRITE")

        print(f"[{i:02d}]  {sec_name:<10} 0x{vaddr:08X}   0x{vsize:08X}   0x{raw_size:08X}   0x{raw_ptr:08X}   {ent:5.2f}    {','.join(flag_desc)}")

        sections.append({
            'Name': sec_name,
            'VirtualAddress': vaddr,
            'VirtualSize': vsize,
            'SizeOfRawData': raw_size,
            'PointerToRawData': raw_ptr,
            'Characteristics': flags,
            'Entropy': ent
        })

    # Find which section contains EntryPoint
    ep_section = None
    for sec in sections:
        if sec['VirtualAddress'] <= entry_point < sec['VirtualAddress'] + max(sec['VirtualSize'], sec['SizeOfRawData']):
            ep_section = sec['Name']
            break
    print(f"\nEntryPoint Location  : RVA 0x{entry_point:08X} falls into section '{ep_section}'")

    # 5. Import Table Inspection
    import_dir = data_dirs.get("IMPORT", {'rva': 0, 'size': 0})
    imports = {}
    if import_dir['rva'] != 0:
        import_offset = rva_to_offset(import_dir['rva'], sections)
        print(f"\n--- Import Table (RVA: 0x{import_dir['rva']:08X}, File Offset: 0x{import_offset:08X}) ---")
        curr_offset = import_offset
        while True:
            orig_first_thunk, timedate, forwarder, name_rva, first_thunk = struct.unpack(
                '<IIIII', data[curr_offset:curr_offset+20]
            )
            if orig_first_thunk == 0 and name_rva == 0 and first_thunk == 0:
                break
            
            dll_name_offset = rva_to_offset(name_rva, sections)
            dll_name = read_cstring(data, dll_name_offset) if dll_name_offset else f"RVA_0x{name_rva:X}"
            
            # Read thunk list
            thunk_rva = orig_first_thunk if orig_first_thunk != 0 else first_thunk
            thunk_offset = rva_to_offset(thunk_rva, sections)
            funcs = []
            
            if thunk_offset:
                t_curr = thunk_offset
                while True:
                    if not is_pe32_plus:
                        entry_val = struct.unpack('<I', data[t_curr:t_curr+4])[0]
                        t_curr += 4
                        if entry_val == 0:
                            break
                        if entry_val & 0x80000000:
                            # Ordinal
                            funcs.append(f"Ordinal_{entry_val & 0xFFFF}")
                        else:
                            name_struct_offset = rva_to_offset(entry_val, sections)
                            if name_struct_offset:
                                hint = struct.unpack('<H', data[name_struct_offset:name_struct_offset+2])[0]
                                fname = read_cstring(data, name_struct_offset + 2)
                                funcs.append(fname)
                    else:
                        entry_val = struct.unpack('<Q', data[t_curr:t_curr+8])[0]
                        t_curr += 8
                        if entry_val == 0:
                            break
                        if entry_val & 0x8000000000000000:
                            funcs.append(f"Ordinal_{entry_val & 0xFFFF}")
                        else:
                            name_struct_offset = rva_to_offset(entry_val, sections)
                            if name_struct_offset:
                                hint = struct.unpack('<H', data[name_struct_offset:name_struct_offset+2])[0]
                                fname = read_cstring(data, name_struct_offset + 2)
                                funcs.append(fname)

            imports[dll_name] = funcs
            curr_offset += 20

        print(f"Total Imported DLLs: {len(imports)}")
        for dll, f_list in imports.items():
            print(f"\n  [DLL] {dll} ({len(f_list)} functions)")
            # Category filters
            net_apis = [f for f in f_list if any(k in f.lower() for k in ['socket', 'connect', 'send', 'recv', 'http', 'internet', 'url', 'gethost', 'bind', 'listen', 'select', 'wsastartup'])]
            file_apis = [f for f in f_list if any(k in f.lower() for k in ['file', 'read', 'write', 'directory', 'disk', 'path'])]
            reg_apis = [f for f in f_list if any(k in f.lower() for k in ['regopen', 'regcreate', 'regquery', 'regset', 'regclose'])]
            ui_apis = [f for f in f_list if any(k in f.lower() for k in ['window', 'dialog', 'message', 'paint', 'menu', 'button', 'edit', 'draw', 'gdi'])]
            proc_apis = [f for f in f_list if any(k in f.lower() for k in ['process', 'thread', 'virtual', 'heap', 'loadlibrary', 'getprocaddress'])]

            if net_apis:
                print(f"    * Network / Socket APIs : {', '.join(net_apis)}")
            if reg_apis:
                print(f"    * Registry APIs         : {', '.join(reg_apis)}")
            if file_apis:
                print(f"    * File / IO APIs        : {', '.join(file_apis[:10])}{' ...' if len(file_apis) > 10 else ''}")
            if ui_apis and len(ui_apis) <= 10:
                print(f"    * UI / Window APIs      : {', '.join(ui_apis)}")
            elif ui_apis:
                print(f"    * UI / Window APIs      : {', '.join(ui_apis[:10])} ... ({len(ui_apis)} total)")

    # 6. Delphi / Borland & Compiler Clues
    print("\n--- Compiler & Technology Stack Fingerprints ---")
    delphi_markers = [
        b'Borland', b'Delphi', b'PACKAGEINFO', b'DVCLAL', b'TApplication',
        b'Forms', b'SysUtils', b'Classes', b'Controls', b'Graphics',
        b'VirtualAlloc', b'MSVBVM', b'MFC42', b'MSVCR', b'Qt5', b'wxWidgets'
    ]
    found_markers = []
    for m in delphi_markers:
        pos = data.find(m)
        if pos != -1:
            found_markers.append((m.decode('latin1'), pos))

    if found_markers:
        print("Signature matches found in binary:")
        for name, pos in found_markers:
            print(f"  - String pattern '{name}' found at offset 0x{pos:X}")

    # Section name heuristics
    sec_names = [s['Name'] for s in sections]
    if 'CODE' in sec_names and 'DATA' in sec_names and 'BSS' in sec_names:
        print("  => Heuristic: CODE / DATA / BSS section naming is classic Borland Delphi / C++Builder!")
    elif '.text' in sec_names and '.rdata' in sec_names and '.data' in sec_names:
        print("  => Heuristic: .text / .rdata / .data is standard Microsoft Visual C++ or MinGW GCC layout.")

    # Check for UPX or common packers
    for s in sections:
        if 'UPX' in s['Name']:
            print(f"  [!] UPX Packer section '{s['Name']}' detected!")
        elif 'ASPack' in s['Name'] or 'PECompact' in s['Name']:
            print(f"  [!] Packer section '{s['Name']}' detected!")

if __name__ == '__main__':
    target = r"C:\software\疯玩神仙道\pie.exe"
    inspect_pe(target)
