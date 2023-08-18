from subprocess import check_output

class Changer:
    def __init__(self, dev, drive_index=0):
        self.dev = dev
        self.drive_index = drive_index

    def eject(self):
        pass

    def read_inventory(self):
        inventory = {}
    
        res = check_output(["mtx", "-f", self.dev, "status"], encoding='utf-8')
        for line in res.splitlines():
            line = line.strip()
            if not line.startswith('Storage Element') and not line.startswith('Data Transfer Element'):
                continue
            sections = line.split(':')
            status = sections[1].strip()
            if status != 'Full':
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

                    inventory[barcode] = index

        return inventory

    def load_by_barcode(self, barcode):
        inventory = self.read_inventory()
        index = inventory[barcode]
        check_output(["mtx", "-f", self.dev, "load", str(index), str(self.drive_index)])
