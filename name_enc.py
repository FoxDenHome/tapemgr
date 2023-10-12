from base64 import urlsafe_b64decode, urlsafe_b64encode
from Crypto.Cipher import AES

class NameCryptor:
    def __init__(self, key: bytes):
        super().__init__()
        self.cipher = AES.new(key, AES.MODE_CBC, iv=b'\x00' * 16)

    def encrypt(self, name: str) -> str:
        return '/'.join(map(self.encrypt_one, name.split('/')))

    def encrypt_one(self, name: str) -> str:
        name_bytes = name.encode('utf-8')
        name_bytes += b'\x00' * (16 - (len(name_bytes) % 16))
        ciphertext = self.cipher.encrypt(name_bytes)

        return urlsafe_b64encode(ciphertext).decode('utf-8')

    def decrypt(self, name: str) -> str:
        return '/'.join(map(self.decrypt_one, name.split('/')))

    def decrypt_one(self, name: str) -> str:
        ciphertext = self.cipher.decrypt(urlsafe_b64decode(name))

        return ciphertext.decode('utf-8').rstrip('\x00')
