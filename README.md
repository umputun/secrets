# Safe Secrets

**Service to share secrets**

The primary use-case - sharing sensetive information by making information self-destructed, accesable once and protected by easy-to-pass pin code.

## Usage [safesecret.info](https://safesecret.info)

## API

### Save message

`POST /v1/message`, body - `{"message":"some top secret info", "exp": 120, "pin": 12345}`
- `exp` expire in N seconds
- `pin` fixed-size ping code 

```
    http POST https://safesecret.info/api/v1/message pin=12345 message=testtest-12345678 exp:=1000
    
    HTTP/1.1 201 Created
    
    {
     "exp": "2016-06-25T13:33:45.11847278-05:00",
     "key": "f1acfe04-277f-4016-518d-16c312ab84b5" 
    }
```

### Load message

`GET /v1/message/:key/:pin`

```
    https://safesecret.info/api/v1/message/6ceab760-3059-4a52-5670-649509b128fc/12345

    HTTP/1.1 200 OK

    {
     "key": "6ceab760-3059-4a52-5670-649509b128fc",
     "message": "testtest-12345678"
    }
```
