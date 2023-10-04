from dataclasses import dataclass
from typing import Sequence
from util import logged_check_output_binary

def scsi_raw(device: str, rlen: int, data: Sequence[int]):
    cmd = ["sg_raw", "-b", "-R", "-r", str(rlen), device]
    for d in data:
        cmd.append("0x%02X" % d)
    return logged_check_output_binary(cmd)

def bool_to_bit(val: bool, bit: int) -> int:
    if val:
        return 1 << bit
    return 0

@dataclass
class SCSIElement:
    index: int
    type_code: int
    flags: int
    data: bytes

    def get_data_transfer_element_identifier(self):
        if self.type_code != 0x04:
            raise ValueError("This is not a data transfer element")
        
        id_len = self.data[59]
        return self.data[60:60+id_len]


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

    first_element = (res[0] << 8) | res[1]
    element_count = (res[2] << 8) | res[3]
    #report_length = (res[5] << 16) | (res[6] << 8) | res[7]

    elements: list[SCSIElement] = []
    pos = 8
    for index in range(element_count):
        print(index, res[pos:])
        data_len = (res[pos+2] << 8) | res[pos+3]

        #descriptor_available = (res[pos+5] << 16) | (res[pos+6] << 8) | (res[pos+7])

        elements.append(SCSIElement(
            index=index + first_element,
            type_code=res[pos],
            flags=res[pos+1],
            data=res[pos+8:pos+8+data_len]
        ))
        pos += 8 + data_len

    return elements
