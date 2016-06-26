# Safe Secrets - easy way to tansfer password

The primary use-case is sharing sensetive data by making this information self-destructed, accesable only once and protected by easy-to-pass pin code. I just needed a simple and better alternative to the most popular way of passing passwords. Doing this by email made me always worry
and the usual "protection" by sending user and password info in two different emails is just a joke.

## Usage

It runs on **[safesecret.info](https://safesecret.info)** for real. Feel free to use it if you crazy enought to trust me,
or just run your own from prepared docker image. Sure, you can build from sources as well.

Create safesecret link to your message by entering 3 things:
 - message itself, like credentials
 - expiration interval
 - pin code

 This will give you a link you may send by email, chat, or transfered by using any other trasnport.
 As soon as recipent opens the link he will be asked for the pin and will get your message.
 Pin is (usually) numeric and easy to pass by voice call or sms.
 Each link can be opened once and number of attempt to enter wrong pin is limited (3 by default).


## How safe is this thing?

- it doesn't keep your original message or pin anythere, but encrypts message with hashed pin
- it doesn't keep anything on disk in any form (in case of InMemory engine)
- it doesn't keep any sensitive info in any logs
- as soon as message read or expired it will be deleted and destoryed completely
- in order to steal your message bad guys will need acces to your link as well as pin code


## Install Secrets

1. Adjust `docker-compose.yml` with:
    - TZ - your local time zone
    - SIGN_KEY - something long and random
    - MAX_EXPIRE - maximum expiration period in secs, default 86400 (24h)
    - PIN_SIZE - default 5
    - PIN_ATTEMPTS - default 3
1. run the system with `docker-compose up -d`. This will download prepared image from docker hub and start all components.
1. if you want to build it from sources - `docker-compose build` will do it, and then `docker-compose up -d`.


## API

Secrets provides trivial REST to save and load messages. Just two calls:

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
