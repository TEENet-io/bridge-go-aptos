# Knowledge

About compressed and uncompressed public key in Bitcoin.

Bitcoin private key is a 256-bit number. It is used to generate a public key. The public key is used to generate a Bitcoin address. The *public key* is a point on the elliptic curve. It is represented by two coordinates (X, Y). One *X* can correspond to two *Y*s. 

For uncompressed public key, the information is full. The X and Y both presents. And final representation includes a 0x04 prefix, X and Y. The uncompressed public key is 65 bytes long.

For compressed public key, the information is half complete. It only contains X. How could that be allowed? The secret lies in the fact that Y can be inferred by X on the elliptic curve. The Y can be calculated by X. The Y can be either even or odd. The compressed public key uses the first byte to indicate the Y is even or odd. If Y is even, the first byte is 0x02. If Y is odd, the first byte is 0x03. The final representation includes the first byte and X. The compressed public key is 33 bytes long.

| Name                    | Size     | Description                        |
|-------------------------|----------|------------------------------------|
| Uncompressed Public Key | 65 bytes | 0x04 + X (32 bytes) + Y (32 bytes) |
| Compressed Public Key   | 33 bytes | 0x02/0x03 + X (32 bytes)           |

Now the compressed version is used more to save tx space on modern Bitcoin, and initially Bitcoin uses uncompressed a lot.