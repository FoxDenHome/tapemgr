from subprocess import check_output

class Changer:
    def __init__(self, dev, drive_index=0):
        self.dev = dev
        self.drive_index = drive_index

    def eject(self):
        pass

    def read_inventory(self):
        inventory = {}
        current_loaded = {}
    
        res = check_output(["mtx", "-f", self.dev, "status"], encoding='utf-8')
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
            status = sections[1].strip()
            if not status.startswith('Full'):
                continue

            index = None

            element_split = sections[0].split(' ')
            for i in range(len(element_split)):
                if element_split[i] == 'Element':
                    index = int(element_split[i+1], 10)
                    break

            for sec in sections[2:]:
                secsplit = sec.split('=')
                if secsplit[0].strip() == 'VolumeTag':
                    barcode = secsplit[1].strip()

                    if index_type == 'storage':
                        inventory[barcode] = index
                    elif index_type == 'drive':
                        current_loaded[index] = barcode

        return inventory, current_loaded

    def load_by_barcode(self, barcode):
        inventory, current_loaded = self.read_inventory()
        if current_loaded.get(self.drive_index, None) == barcode:
            return
        index = inventory[barcode]
        check_output(["mtx", "-f", self.dev, "load", str(index), str(self.drive_index)])

    def read_label(self):
        _, current_loaded = self.read_inventory()
        return current_loaded.get(self.drive_index, None)
