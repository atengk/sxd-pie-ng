"""神仙道传统辅助 (pie.exe) 专属网络流量监听与实时抓包分析工具。

支持动态嗅探进程 TCP 长连接、自动同步配置 pktmon 硬件级抓包筛选器、
导出标准 Wireshark pcapng 格式，并内置神仙道私有二进制协议解码器。

@author Ateng
@since 2026-10-09
"""

import argparse
import ctypes
from ctypes import wintypes
import datetime
import os
import shutil
import socket
import struct
import subprocess
import sys
import time

# 核心常量定义
DEFAULT_PIE_PATH = r"C:\software\疯玩神仙道\pie.exe"
DEFAULT_GAME_PORT = 8381
OUTPUT_PCAP_PATH = r"d:\sources\my\sxd-pie-ng\sxd.pcapng"
TEMP_ETL_PATH = r"C:\sxd_capture.etl"

# Win32 API 常量
TH32CS_SNAPPROCESS = 0x00000002
AF_INET = 2
TCP_TABLE_OWNER_PID_ALL = 5
PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

# MIB_TCP_STATE 枚举
TCP_STATE_MAP = {
    1: "CLOSED",
    2: "LISTEN",
    3: "SYN_SENT",
    4: "SYN_RCVD",
    5: "ESTABLISHED",
    6: "FIN_WAIT1",
    7: "FIN_WAIT2",
    8: "CLOSE_WAIT",
    9: "CLOSING",
    10: "LAST_ACK",
    11: "TIME_WAIT",
    12: "DELETE_TCB",
}


class PROCESSENTRY32W(ctypes.Structure):
    _fields_ = [
        ("dwSize", wintypes.DWORD),
        ("cntUsage", wintypes.DWORD),
        ("th32ProcessID", wintypes.DWORD),
        ("th32DefaultHeapID", ctypes.c_size_t),
        ("th32ModuleID", wintypes.DWORD),
        ("cntThreads", wintypes.DWORD),
        ("th32ParentProcessID", wintypes.DWORD),
        ("pcPriClassBase", wintypes.LONG),
        ("dwFlags", wintypes.DWORD),
        ("szExeFile", wintypes.WCHAR * 260),
    ]


class MIB_TCPROW_OWNER_PID(ctypes.Structure):
    _fields_ = [
        ("dwState", wintypes.DWORD),
        ("dwLocalAddr", wintypes.DWORD),
        ("dwLocalPort", wintypes.DWORD),
        ("dwRemoteAddr", wintypes.DWORD),
        ("dwRemotePort", wintypes.DWORD),
        ("dwOwningPid", wintypes.DWORD),
    ]


kernel32 = ctypes.windll.kernel32
iphlpapi = ctypes.windll.iphlpapi
shell32 = ctypes.windll.shell32


def is_admin() -> bool:
    """检查当前 Python 进程是否拥有管理员特权。

    @return bool 是否拥有管理员权限
    """
    try:
        return bool(shell32.IsUserAnAdmin())
    except Exception:
        return False


def elevate_privileges():
    """触发 Windows UAC 提权并唤起管理员终端执行本脚本。"""
    print("[*] pktmon 驱动需要管理员特权，正在请求 UAC 提权启动...")
    script_abs = os.path.abspath(sys.argv[0])
    args_list = [f'"{script_abs}"'] + [f'"{arg}"' for arg in sys.argv[1:]]
    params = " ".join(args_list)
    script_dir = os.path.dirname(script_abs)
    ret = shell32.ShellExecuteW(None, "runas", sys.executable, params, script_dir, 1)
    if ret <= 32:
        print(f"[-] UAC 提权被取消或失败，错误代码: {ret}")
        sys.exit(1)
    print("[+] 管理员窗口已成功启动，当前引导进程退出。")
    sys.exit(0)


def find_processes_by_name(target_name: str = "pie.exe") -> list:
    """遍历系统快照获取指定可执行文件名对应的所有进程列表。

    @param target_name 目标可执行文件名
    @return list 匹配进程元组列表 [(pid, exe_name, full_path)]
    """
    h_snap = kernel32.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
    if h_snap == -1:
        return []

    pe = PROCESSENTRY32W()
    pe.dwSize = ctypes.sizeof(PROCESSENTRY32W)
    results = []

    if kernel32.Process32FirstW(h_snap, ctypes.byref(pe)):
        while True:
            if pe.szExeFile.lower() == target_name.lower():
                full_path = ""
                h_proc = kernel32.OpenProcess(
                    PROCESS_QUERY_LIMITED_INFORMATION, False, pe.th32ProcessID
                )
                if h_proc:
                    buf = (wintypes.WCHAR * 1024)()
                    size = wintypes.DWORD(1024)
                    if kernel32.QueryFullProcessImageNameW(
                        h_proc, 0, buf, ctypes.byref(size)
                    ):
                        full_path = buf.value
                    kernel32.CloseHandle(h_proc)
                results.append((pe.th32ProcessID, pe.szExeFile, full_path))
            if not kernel32.Process32NextW(h_snap, ctypes.byref(pe)):
                break
    kernel32.CloseHandle(h_snap)
    return results


def get_tcp_connections_for_pid(target_pid: int) -> list:
    """基于 Windows ExtendedTcpTable 查询指定 PID 的所有 TCP 连接条目。

    @param target_pid 目标进程 PID
    @return list 连接信息字典列表
    """
    table_size = wintypes.DWORD(0)
    iphlpapi.GetExtendedTcpTable(
        None, ctypes.byref(table_size), True, AF_INET, TCP_TABLE_OWNER_PID_ALL, 0
    )
    if table_size.value == 0:
        return []

    buf = (ctypes.c_byte * table_size.value)()
    ret = iphlpapi.GetExtendedTcpTable(
        ctypes.byref(buf),
        ctypes.byref(table_size),
        True,
        AF_INET,
        TCP_TABLE_OWNER_PID_ALL,
        0,
    )
    if ret != 0:
        return []

    num_entries = ctypes.cast(buf, ctypes.POINTER(wintypes.DWORD))[0]
    row_size = ctypes.sizeof(MIB_TCPROW_OWNER_PID)
    rows_start = ctypes.addressof(buf) + 4

    conns = []
    for i in range(num_entries):
        row = MIB_TCPROW_OWNER_PID.from_address(rows_start + i * row_size)
        if row.dwOwningPid == target_pid:
            local_ip = socket.inet_ntoa(
                ctypes.c_uint32(row.dwLocalAddr).value.to_bytes(4, "little")
            )
            local_port = socket.ntohs(row.dwLocalPort)
            remote_ip = socket.inet_ntoa(
                ctypes.c_uint32(row.dwRemoteAddr).value.to_bytes(4, "little")
            )
            remote_port = socket.ntohs(row.dwRemotePort)
            state_str = TCP_STATE_MAP.get(row.dwState, f"STATE_{row.dwState}")
            conns.append(
                {
                    "pid": row.dwOwningPid,
                    "state": state_str,
                    "local": f"{local_ip}:{local_port}",
                    "remote": f"{remote_ip}:{remote_port}",
                    "local_ip": local_ip,
                    "local_port": local_port,
                    "remote_ip": remote_ip,
                    "remote_port": remote_port,
                }
            )
    return conns


def run_command(cmd: list) -> tuple:
    """执行底层控制台指令并捕获标准输出与标准错误。

    @param cmd 待执行命令行数组
    @return tuple (exit_code, stdout, stderr)
    """
    proc = subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="gbk",
        errors="ignore",
    )
    return proc.returncode, proc.stdout, proc.stderr


def log(msg: str):
    """同时输出到控制台与审计日志文件。"""
    print(msg)
    try:
        log_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "capture.log")
        with open(log_path, "a", encoding="utf-8") as f:
            ts = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            f.write(f"[{ts}] {msg}\n")
    except Exception:
        pass


def configure_and_start_pktmon(ports: set) -> bool:
    """配置 pktmon 硬件过滤规则并启动实时抓包会话。

    // 1. 清理存量抓包规则与会话残余
    // 2. 注入目标端口精确过滤名单
    // 3. 启动全帧长无损捕获 (pkt-size 0)

    @param ports 待抓包目标端口集合
    @return bool 启动是否成功
    """
    log("[*] 正在初始化 pktmon 过滤规则...")
    run_command(["pktmon", "stop"])
    run_command(["pktmon", "filter", "remove"])

    # 注入端口过滤规则 (与用户成功验证的 pktmon filter add -p <port> 严格对齐)
    for p in ports:
        code, out, err = run_command(
            ["pktmon", "filter", "add", "-p", str(p)]
        )
        if code == 0:
            log(f"[+] 已装载端口过滤器: TCP Port {p}")
        else:
            log(f"[-] 装载端口 {p} 失败: {err.strip() or out.strip()}")

    # 启动抓包会话
    log(f"[*] 正在启动抓包引擎，输出暂存文件: {TEMP_ETL_PATH}")
    code, out, err = run_command(
        ["pktmon", "start", "-c", "--pkt-size", "0", "-f", TEMP_ETL_PATH]
    )
    if code != 0:
        log(f"[-] 启动 pktmon 抓包引擎失败: {err.strip() or out.strip()}")
        return False
    log("[+] pktmon 抓包引擎已就绪并全速运行中！")
    return True


def stop_pktmon_and_export(output_pcap: str) -> bool:
    """停止 pktmon 会话并将 etl 数据集转换为标准 pcapng 格式。

    // 1. 发出终止信号停止数据采集
    // 2. 调用 etl2pcap 转换二进制格式
    // 3. 校验并归档生成的 pcapng 文件

    @param output_pcap 目标 pcapng 文件落盘路径
    @return bool 导出是否成功
    """
    print("\n[*] 正在停止 pktmon 抓包会话...")
    run_command(["pktmon", "stop"])

    if not os.path.exists(TEMP_ETL_PATH):
        print(f"[-] 未找到临时追踪文件: {TEMP_ETL_PATH}")
        return False

    print(f"[*] 正在将 ETL 转换为标准 Wireshark pcapng: {output_pcap}")
    os.makedirs(os.path.dirname(os.path.abspath(output_pcap)), exist_ok=True)
    code, out, err = run_command(
        ["pktmon", "etl2pcap", TEMP_ETL_PATH, "-o", output_pcap]
    )
    if code != 0 or not os.path.exists(output_pcap):
        print(f"[-] 转换 pcapng 失败: {err.strip() or out.strip()}")
        return False

    file_size = os.path.getsize(output_pcap)
    print(f"[+] 抓包归档成功: {output_pcap} ({file_size} 字节)")

    # 复制一份时间戳历史归档
    ts = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
    archive_dir = os.path.join(os.path.dirname(output_pcap), "data", "captures")
    os.makedirs(archive_dir, exist_ok=True)
    archive_path = os.path.join(archive_dir, f"pie_capture_{ts}.pcapng")
    shutil.copyfile(output_pcap, archive_path)
    print(f"[+] 历史归档已保存至: {archive_path}")
    return True


def parse_and_summarize_pcapng(pcap_path: str):
    """解析生成的 pcapng 文件，深度提取并打印神仙道游戏二进制封包协议明细。

    // 1. 遍历 pcapng 的 EPB 封包块并提取 IPv4 TCP 报文
    // 2. 解包 TCP 载荷并依据 [4B Length][4B ActionID] 解构神仙道私有帧
    // 3. 输出汇总统计与关键指令摘要列表

    @param pcap_path pcapng 文件路径
    """
    if not os.path.exists(pcap_path):
        return

    print("\n==================== 封包协议深度分析报告 ====================")
    with open(pcap_path, "rb") as f:
        data = f.read()

    pos = 0
    total_packets = 0
    sxd_frames = []

    while pos < len(data):
        if pos + 8 > len(data):
            break
        block_type, block_len = struct.unpack("<II", data[pos : pos + 8])
        if block_len < 12 or pos + block_len > len(data):
            break

        if block_type == 6:  # Enhanced Packet Block
            caplen, _ = struct.unpack("<II", data[pos + 20 : pos + 28])
            pkt = data[pos + 28 : pos + 28 + caplen]
            if len(pkt) >= 14:
                eth_type = struct.unpack("!H", pkt[12:14])[0]
                if eth_type == 0x0800 and len(pkt) >= 34:  # IPv4
                    ip_hdr = pkt[14:]
                    proto = ip_hdr[9]
                    ihl = (ip_hdr[0] & 0x0F) * 4
                    src_ip = ".".join(str(b) for b in ip_hdr[12:16])
                    dst_ip = ".".join(str(b) for b in ip_hdr[16:20])

                    if proto == 6 and len(ip_hdr) >= ihl + 20:  # TCP
                        tcp_hdr = ip_hdr[ihl:]
                        src_port, dst_port = struct.unpack("!HH", tcp_hdr[0:4])
                        tcp_offset = ((tcp_hdr[12] >> 4) & 0x0F) * 4
                        payload = tcp_hdr[tcp_offset:]
                        total_packets += 1

                        if payload:
                            # 识别神仙道封包
                            direction = (
                                "UP (客户端->服务端)"
                                if dst_port in (DEFAULT_GAME_PORT, 19753)
                                else "DOWN (服务端->客户端)"
                            )
                            if len(payload) >= 8:
                                pkt_len, action_id = struct.unpack(
                                    "!II", payload[:8]
                                )
                                mod = action_id >> 16
                                act = action_id & 0xFFFF
                                is_zlib = len(payload) > 8 and payload[
                                    8:10
                                ] in (b"\x78\x9c", b"\x78\xda", b"\x78\x01")
                                sxd_frames.append(
                                    {
                                        "direction": direction,
                                        "src": f"{src_ip}:{src_port}",
                                        "dst": f"{dst_ip}:{dst_port}",
                                        "payload_len": len(payload),
                                        "sxd_len": pkt_len,
                                        "action_id": action_id,
                                        "module": mod,
                                        "action": act,
                                        "zlib": is_zlib,
                                    }
                                )
        pos += block_len

    print(
        f"抓包总 TCP 报文数: {total_packets} 个 | 解析有效神仙道私有封包: {len(sxd_frames)} 个"
    )
    print("---------------------------------------------------------------")
    for i, frame in enumerate(sxd_frames[:25]):
        zlib_tag = " [ZLIB 压缩]" if frame["zlib"] else ""
        print(
            f"[{i+1:02d}] {frame['direction']:<18} | 模块: {frame['module']:<4} 动作: {frame['action']:<4} | "
            f"ActionID: 0x{frame['action_id']:08X} | 声明长度: {frame['sxd_len']:<5} 实际载荷: {frame['payload_len']}{zlib_tag}"
        )
    if len(sxd_frames) > 25:
        print(f"... 其余 {len(sxd_frames) - 25} 个封包已保存至 pcapng 文件中。")
    print("===============================================================\n")


def main():
    parser = argparse.ArgumentParser(
        description="神仙道 pie.exe 专属网络监听与抓包器"
    )
    parser.add_argument(
        "--output",
        "-o",
        default=OUTPUT_PCAP_PATH,
        help=f"生成的 pcapng 保存路径 (默认: {OUTPUT_PCAP_PATH})",
    )
    parser.add_argument(
        "--exe",
        "-e",
        default=DEFAULT_PIE_PATH,
        help=f"目标 pie.exe 路径 (默认: {DEFAULT_PIE_PATH})",
    )
    parser.add_argument(
        "--duration",
        "-d",
        type=int,
        default=0,
        help="自动抓包持续秒数 (0 表示持续监听直至手动中断)",
    )
    parser.add_argument(
        "--monitor-only",
        "-m",
        action="store_true",
        help="仅监听进程与连接变化，不启动底层 pktmon 抓包驱动 (无需管理员权限)",
    )
    args = parser.parse_args()

    log("==================================================")
    log("      神仙道 pie.exe 专属网络监听与自动抓包器       ")
    log("==================================================")

    # 1. 管理员特权校验与自适应降级
    if not is_admin():
        if not args.monitor_only:
            log("[!] 当前为普通用户权限 (非管理员)。")
            log("    - 若需硬件级 pktmon 原始数据包捕获生成 pcapng，请右键 scripts\\capture_pie.bat「以管理员身份运行」。")
            log("[*] 当前自动启动进程长连接实时监听与状态嗅探模式...")
            args.monitor_only = True
    else:
        log("[+] 已检测到管理员特权，pktmon 驱动级无损抓包引擎已就绪。")

    # 2. 检查或定位 pie.exe 目标进程
    target_procs = find_processes_by_name("pie.exe")
    target_pid = None
    if target_procs:
        target_pid, exe_name, full_path = target_procs[0]
        log(f"[+] 成功捕获目标进程: {exe_name} (PID: {target_pid})")
        log(f"    可执行文件路径: {full_path or args.exe}")
    else:
        log(f"[!] 未检测到运行中的 pie.exe，正在尝试启动: {args.exe}")
        if os.path.exists(args.exe):
            workdir = os.path.dirname(args.exe)
            proc = subprocess.Popen([args.exe], cwd=workdir)
            target_pid = proc.pid
            time.sleep(1.5)
            log(f"[+] 已启动 pie.exe (PID: {target_pid})")
        else:
            log(f"[-] 目标文件不存在: {args.exe}")
            log("[*] 将进入待命监听模式，持续轮询等待 pie.exe 进程上线...")

    # 3. 嗅探网络端口并动态装载抓包过滤名单
    active_ports = {DEFAULT_GAME_PORT, 19753}
    if target_pid:
        conns = get_tcp_connections_for_pid(target_pid)
        for c in conns:
            if c["remote_port"] > 0:
                active_ports.add(c["remote_port"])
            if c["local_port"] > 0 and c["state"] == "LISTEN":
                active_ports.add(c["local_port"])
        log(f"[*] pie.exe 当前活跃连接: {len(conns)} 条")
        for c in conns:
            log(f"    - {c['local']} -> {c['remote']} [{c['state']}]")

    log(f"[*] 纳入抓包的监控端口集: {sorted(list(active_ports))}")

    # 4. 启动 pktmon 抓包引擎 (非监控模式下)
    if not args.monitor_only:
        if not configure_and_start_pktmon(active_ports):
            log("[-] 抓包引擎启动失败，流程中止。")
            if sys.stdin and sys.stdin.isatty():
                try:
                    input("\n[!] 按回车键退出窗口...")
                except Exception:
                    pass
            return
        log("\n[+] 抓包监听已全面启动！")
        log("    正在实时监听 pie.exe 的网络通信与游戏交互...")
    else:
        log("\n[+] 纯监听巡检模式已启动 (不调用驱动抓包)...")
    if args.duration > 0:
        log(f"    将在 {args.duration} 秒后自动完成并归档。")
    else:
        log("    按 Ctrl+C、空格或输入 'q' / Enter 可随时停止抓包并生成报告。\n")

    start_time = time.time()
    last_conn_signatures = set()

    try:
        while True:
            # 持续巡检 pie.exe 进程与连接状态
            current_procs = find_processes_by_name("pie.exe")
            if current_procs:
                curr_pid = current_procs[0][0]
                conns = get_tcp_connections_for_pid(curr_pid)
                curr_signatures = {
                    f"{c['local']}->{c['remote']}:{c['state']}" for c in conns
                }
                new_conns = curr_signatures - last_conn_signatures
                if new_conns and last_conn_signatures:
                    for sig in new_conns:
                        log(
                            f"[{datetime.datetime.now().strftime('%H:%M:%S')}] pie.exe 新连接变化: {sig}"
                        )
                last_conn_signatures = curr_signatures

            # 支持按回车键或 'q' 键手动结束抓包
            try:
                import msvcrt
                if msvcrt.kbhit():
                    ch = msvcrt.getch()
                    if ch in (b'\r', b'\n', b'q', b'Q', b' '):
                        log("\n[*] 接收到用户键盘输入，正在停止抓包...")
                        break
            except Exception:
                pass

            # 检查自动停止计时
            if args.duration > 0 and (time.time() - start_time) >= args.duration:
                log(f"\n[*] 抓包时长已达到预设的 {args.duration} 秒。")
                break

            time.sleep(0.5)

    except KeyboardInterrupt:
        log("\n[*] 接收到用户中断信号 (Ctrl+C)...")

    # 5. 导出 pcapng 并输出解码报告
    if not args.monitor_only:
        if stop_pktmon_and_export(args.output):
            parse_and_summarize_pcapng(args.output)
    else:
        log("[+] 巡检监听结束。")

    if sys.stdin and sys.stdin.isatty():
        try:
            input("\n[+] 抓包分析已完成，按回车键 (Enter) 退出窗口...")
        except Exception:
            pass


if __name__ == "__main__":
    main()
