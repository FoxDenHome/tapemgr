from util import logged_check_call, logged_check_output
from dataclasses import dataclass
from typing import Literal, Optional

@dataclass
class Slot:
    attributes: set[str]
    type: Literal['storage', 'drive'] = 'storage'
    index: int = -1
    barcode: str = ""
    empty: bool = False


class Changer:
    def __init__(self, dev: str, drive_index: int):
        super().__init__()
        self.dev = dev
        self.drive_index = drive_index

    def eject(self) -> None:
        logged_check_call(['mtx', '-f', self.dev, 'eject'])

    def read_inventory(self) -> tuple[dict[int, Slot], dict[int, Slot]]:
        drive_inventory: dict[int, Slot] = {}
        storage_inventory: dict[int, Slot] = {}

        res = logged_check_output(['mtx', '-f', self.dev, 'status'])
        for line in res.splitlines():
            line = line.strip()

            slot = Slot(attributes=set())
            if line.startswith('Storage Element'):
                slot.type = 'storage'
            elif line.startswith('Data Transfer Element'):
                slot.type = 'drive'
            else:
                continue
            sections = line.split(':')

            element_split = sections[0].split(' ')
            for i in range(len(element_split)):
                if element_split[i] == 'Element':
                    slot.index = int(element_split[i+1], 10)
                    slot.attributes = set(element_split[i+2:])
                    break

            if slot.index < 0:
                continue

            if slot.type == 'storage':
                storage_inventory[slot.index] = slot
            elif slot.type == 'drive':
                drive_inventory[slot.index] = slot

            status = sections[1].strip()
            if not status.startswith('Full'):
                if status.startswith('Empty'):
                    slot.empty = True
                continue

            for sec in sections[2:]:
                secsplit = sec.split('=')
                if secsplit[0].strip() == 'VolumeTag':
                    slot.barcode = secsplit[1].strip()

        return storage_inventory, drive_inventory

    def _find_first_empty(self, inventory: dict[int, Slot]) -> Optional[Slot]:
        for slot in inventory.values():
            if not slot.empty:
                continue
            if 'IMPORT/EXPORT' in slot.attributes:
                continue
            return slot
        return None

    def _find_by_barcode(self, barcode: str, inventory: dict[int, Slot]) -> Optional[Slot]:
        for slot in inventory.values():
            if slot.empty:
                continue
            if slot.barcode != barcode:
                continue
            return slot
        return None

    def _unload_slot(self, drive_slot: Slot, storage_inventory: dict[int, Slot]) -> None:
        empty_slot = self._find_first_empty(storage_inventory)
        if not empty_slot:
            raise ValueError('No empty storage slot found')
        logged_check_call(['mtx', '-f', self.dev, 'unload', str(empty_slot), str(drive_slot.index)])

    def unload_current(self) -> None:
        storage_inventory, drive_inventory = self.read_inventory()
        drive_slot = drive_inventory[self.drive_index]

        if drive_slot.empty:
            return

        self._unload_slot(drive_slot, storage_inventory)

    def load_by_barcode(self, barcode: str) -> None:
        storage_inventory, drive_inventory = self.read_inventory()
        drive_slot = drive_inventory[self.drive_index]

        if drive_slot.barcode == barcode:
            return

        self._unload_slot(drive_slot, storage_inventory)

        storage_slot = self._find_by_barcode(barcode, storage_inventory)
        if not storage_slot:
            raise ValueError(f'No storage slot found with barcode {barcode}')

        logged_check_call(['mtx', '-f', self.dev, 'load', str(storage_slot.index), str(drive_slot.index)])

    def read_barcode(self) -> str:
        _, drive_inventory = self.read_inventory()
        return drive_inventory[self.drive_index].barcode
