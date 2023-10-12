from base64 import urlsafe_b64decode, urlsafe_b64encode
from Crypto.Cipher import AES

def encrypt_filename(name: str, key: bytes) -> str:
    cipher = AES.new(key, AES.MODE_CBC, iv=b'\x00' * 16)
    ciphertext = cipher.encrypt(name.encode('utf-8'))
    return urlsafe_b64encode(ciphertext).decode('utf-8')

def decrypt_filename(name: str, key: bytes) -> str:
    cipher = AES.new(key, AES.MODE_CBC, iv=b'\x00' * 16)
    ciphertext = cipher.decrypt(urlsafe_b64decode(name))
    return ciphertext.decode('utf-8').rstrip('\x00')
