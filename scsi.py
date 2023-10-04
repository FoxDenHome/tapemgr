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

    def has_pvol_tag(self) -> bool:
        return (self.flags & 0b10000000) != 0

    def has_avol_tag(self) -> bool:
        return (self.flags & 0b01000000) != 0

    def get_dte_identifier(self) -> str:
        if self.type_code != 0x04:
            raise ValueError("This is not a data transfer element")
        
        base_pos = 15
        if self.has_pvol_tag():
            base_pos += 36
        id_len = self.data[base_pos]
        return self.data[base_pos+1:base_pos+1+id_len].decode("utf-8")
    
    def get_dte_vendor(self) -> str:
        return self.get_dte_identifier()[:8].strip()

    def get_dte_model(self) -> str:
        return self.get_dte_identifier()[8:24].strip()

    def get_dte_serial(self) -> str:
        return self.get_dte_identifier()[24:].strip()

    def compute_properties(self) -> None:
        self.index = (self.data[0] << 8) | self.data[1]
        if self.type_code == 0x04: # Drive / Data Transfer Element start at 0, actually
            self.index -= 1

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
    #element_count = (res[2] << 8) | res[3]
    report_length = (res[5] << 16) | (res[6] << 8) | res[7]

    elements: list[SCSIElement] = []
    pos = 8
    while pos < report_length + 8:
        element_len = (res[pos+2] << 8) | res[pos+3]
        descriptor_len = (res[pos+5] << 16) | (res[pos+6] << 8) | (res[pos+7])

        element_type_code = res[pos]
        element_flags = res[pos+1]

        pos += 8

        sub_pos = 0
        while sub_pos < descriptor_len:
            new_element = SCSIElement(
                index=len(elements) + first_element,
                type_code=element_type_code,
                flags=element_flags,
                data=res[pos+sub_pos:pos+sub_pos+element_len]
            )
            new_element.compute_properties()
            elements.append(new_element)

            sub_pos += element_len

        pos += descriptor_len

    return elements
