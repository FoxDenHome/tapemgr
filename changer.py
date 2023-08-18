from subprocess import check_output

class Changer:
    def __init__(self, dev):
        self.dev = dev

    def read_inventory(self):
        res = check_output(["mtx", "-f", self.dev, "status"])
        for line in res.splitlines():
            line = line.strip()
            if not line.startswith('Storage Element') and not line.startswith('Data Transfer Element'):
                continue
            sections = line.split(':')
            status = sections[1].strip()
            if status != 'Full':
                continue
            print(sections)

    def load_by_barcode(self, barcode):
        self.read_inventory()

c = Changer('/dev/sch0')
c.read_inventory()
