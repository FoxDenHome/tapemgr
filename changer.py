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

IO_BAY_ATTRIBUTE = 'IMPORT/EXPORT'

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

    def _find_first_iobay(self, inventory: dict[int, Slot]) -> Optional[Slot]:
        for slot in inventory.values():
            if IO_BAY_ATTRIBUTE not in slot.attributes:
                continue
            return slot
        return None

    def _find_first_empty(self, inventory: dict[int, Slot]) -> Optional[Slot]:
        return self._find_first_by(inventory, empty=True, io_bay=False)

    def _find_first_by(self, inventory: dict[int, Slot], empty: bool, io_bay: bool) -> Optional[Slot]:
        for slot in inventory.values():
            if empty != slot.empty:
                continue
            if (IO_BAY_ATTRIBUTE in slot.attributes) != io_bay:
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
        if drive_slot.empty:
            return

        empty_slot = self._find_first_empty(storage_inventory)
        if not empty_slot:
            raise ValueError('No empty storage slot found')
        logged_check_call(['mtx', '-f', self.dev, 'unload', str(empty_slot.index), str(drive_slot.index)])

    def _load_slot(self, drive_slot: Slot, storage_slot: Slot, storage_inventory: dict[int, Slot]) -> None:
        self._unload_slot(drive_slot, storage_inventory)
        logged_check_call(['mtx', '-f', self.dev, 'load', str(storage_slot.index), str(drive_slot.index)])


    def unload_current(self) -> None:
        storage_inventory, drive_inventory = self.read_inventory()
        drive_slot = drive_inventory[self.drive_index]
        self._unload_slot(drive_slot, storage_inventory)

    def load_by_barcode(self, barcode: str) -> None:
        storage_inventory, drive_inventory = self.read_inventory()
        drive_slot = drive_inventory[self.drive_index]

        if drive_slot.barcode == barcode:
            return

        storage_slot = self._find_by_barcode(barcode, storage_inventory)
        if not storage_slot:
            raise ValueError(f'No storage slot found with barcode {barcode}')

        self._load_slot(drive_slot, storage_slot, storage_inventory)

    def read_barcode(self) -> str:
        _, drive_inventory = self.read_inventory()
        return drive_inventory[self.drive_index].barcode

    def import_from_iobay(self) -> None:
        storage_inventory, _ = self.read_inventory()

        iobay_slot = self._find_first_by(storage_inventory, empty=False, io_bay=True)
        if not iobay_slot:
            raise ValueError('No loaded IO bay found')

        empty_slot = self._find_first_empty(storage_inventory)
        if not empty_slot:
            raise ValueError('No empty storage slot found')
        
        logged_check_call(['mtx', '-f', self.dev, 'transfer', str(iobay_slot.index), str(empty_slot.index)])

    def export_to_iobay_by_barcode(self, barcode: str) -> None:
        storage_inventory, _ = self.read_inventory()

        iobay_slot = self._find_first_by(storage_inventory, empty=True, io_bay=True)
        if not iobay_slot:
            raise ValueError('No empty IO bay found')

        storage_slot = self._find_by_barcode(barcode, storage_inventory)
        if not storage_slot:
            raise ValueError(f'No storage slot found with barcode {barcode}')

        logged_check_call(['mtx', '-f', self.dev, 'transfer', str(storage_slot.index), str(iobay_slot.index)])
