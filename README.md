# Safe Secrets

**Service to share secrets**

The primary use-case - sharing sensetive information by making information self-destructed, accesable once and protected by easy-to-pass pin code.

## Usage [safesecret.info](https://safesecret.info)

## API

- Save message: `POST /v1/message`, body - `{"message":"some top secret info", "exp": 120, "pin": 12345}`
    - `exp` expire in N seconds
    - `pin` fixed-size ping code
  
