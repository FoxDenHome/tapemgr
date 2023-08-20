from util import logged_check_call, logged_check_output

class Changer:
    def __init__(self, dev, drive_index):
        self.dev = dev
        self.drive_index = drive_index

    def eject(self):
        logged_check_call(["mtx", "-f", self.dev, "eject"])

    def read_inventory(self):
        inventory = {}
        current_loaded = {}
        empty_slots = []
    
        res = logged_check_output(["mtx", "-f", self.dev, "status"], encoding='utf-8')
        for line in res.splitlines():
            line = line.strip()
            index_type = None
            if line.startswith('Storage Element'):
                index_type = 'storage'
            elif line.startswith('Data Transfer Element'):
                index_type = 'drive'
            else:
                continue
            sections = line.split(':')

            index = None

            element_split = sections[0].split(' ')
            for i in range(len(element_split)):
                if element_split[i] == 'Element':
                    index = int(element_split[i+1], 10)
                    break

            status = sections[1].strip()
            if not status.startswith('Full'):
                if status.startswith('Empty'):
                    empty_slots.append(index)
                continue

            for sec in sections[2:]:
                secsplit = sec.split('=')
                if secsplit[0].strip() == 'VolumeTag':
                    barcode = secsplit[1].strip()

                    if index_type == 'storage':
                        inventory[barcode] = index
                    elif index_type == 'drive':
                        current_loaded[index] = barcode

        return inventory, current_loaded, empty_slots

    def unload_current(self):
        _, current_loaded, empty_slots = self.read_inventory()

        current_tape = current_loaded.get(self.drive_index, None)
        if current_tape:
            logged_check_call(["mtx", "-f", self.dev, "unload", str(empty_slots[0]), str(self.drive_index)])

    def load_by_barcode(self, barcode):
        inventory, current_loaded, empty_slots = self.read_inventory()

        current_tape = current_loaded.get(self.drive_index, None)
        if current_tape == barcode:
            return
        
        if current_tape:
            logged_check_call(["mtx", "-f", self.dev, "unload", str(empty_slots[0]), str(self.drive_index)])

        index = inventory[barcode]
        logged_check_call(["mtx", "-f", self.dev, "load", str(index), str(self.drive_index)])

    def read_barcode(self):
        _, current_loaded, _ = self.read_inventory()
        return current_loaded.get(self.drive_index, None)
