# Safe Secrets - easy way to tansfer password

The primary use-case - sharing sensetive information by making this info self-destructed, accesable only once and protected by easy-to-pass pin code.
I just needed a simple and better alternative to the most popular way of passing passwords. Doing this by email made me always worry
and the usual "protection" by sending user and password info in two different emails is just a joke.

## Usage 

It runs on **[safesecret.info](https://safesecret.info)** for real. Feel free to use it if you crazy enought to trust me,
or just run your own from prepared docker image. Sure, you can build from sources as well.

Create safesecret link to your message by entering 3 things:
 - message itself, like credentials
 - expiration interval
 - pin code
 
 This will give you a link you may send by email, chat, or by using any other trasnport. 
 As soon as recipent opens the link he/she will be asked for pin and will get your message. 
 Pin is (usually) numeric, and easy to pass by phone call or sms.
 Each link can be opened once, and number of attempt to enter wrong pin is limited (3 by default).
 
 
## How safe is this thing?

- it doesn't keep anything on disk in any form
- it doesn't keep your messages or pin in memory, but encrypts message with pin and hashes pin
- it doesn't keep any sensitive info you pass in any log
- as soon as message read or expired it will be deleted completely


## API

Secrets provides simple REST to save and load messages. Just two easy calls:

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
