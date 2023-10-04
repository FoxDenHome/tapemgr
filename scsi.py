from typing import Sequence
from util import logged_check_output

def scsi_raw(device: str, rlen: int, data: Sequence[int]):
    cmd = ["sg_raw", "-R", "-r", str(rlen), device]
    for d in data:
        cmd.append("0x%02X" % d)
    return logged_check_output(cmd, encoding=None)

def bool_to_bit(val: bool, bit: int) -> int:
    if val:
        return 1 << bit
    return 0

def scsi_read_element_status(device: str, lun: int, vol_tag: bool, element_type_code: int, start: int, count: int, dont_move: bool, device_id: bool):
    rlen = 1000
    res = scsi_raw(device, rlen, [
        0xB8,
        (lun << 5) | bool_to_bit(vol_tag, 4) | element_type_code,
        start >> 8, start & 0xFF,
        count >> 8, count & 0xFF,
        bool_to_bit(dont_move, 1) | bool_to_bit(device_id, 0),
        (rlen >> 16),
        (rlen >> 8) & 0xFF,
        rlen & 0xFF,
        0x00,
        0x00,
    ])

    print(res)
