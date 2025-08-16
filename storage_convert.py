from storage import Storage
from sys import argv

store = Storage(argv[1])
store.load_all()
store.save_all()

