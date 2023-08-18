from subprocess import check_output

class Changer:
    def __init__(self, dev):
        self.dev = dev

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

            index = int(sections[0].split(' ')[-1], 10)

            for sec in sections[2:]:
                secsplit = sec.split('=')
                if secsplit[0].strip() == 'VolumeTag':
                    barcode = secsplit[1].strip()
                    print(index, barcode)


    def load_by_barcode(self, barcode):
        self.read_inventory()

c = Changer('/dev/sch0')
c.read_inventory()
