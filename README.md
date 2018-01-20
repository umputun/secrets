# Safe Secrets - safe(r) and easy way to transfer passwords

The primary use-case is sharing sensitive data by making the information self-destructed, accessible only once and protected
by easy-to-share PIN code. I just needed a simple and better alternative to the most popular way of passing passwords,
which is why this project was created. Doing this by email always made me concerned about the usual "security" of sending user
and password info in two different emails - which is just a joke.

## Usage

It runs on **[safesecret.info](https://safesecret.info)** for real. Feel free to use it if you are crazy enough to trust me,
or just run your own from prepared docker image. And of course, you can build from sources as well.

Create a **safesecret** link to your message by entering 3 things:

- Content of your secret message
- Expiration time of your secret message
- Secret PIN

 This will give you a link which you can send by email, chat or share by using any other means.
 As soon as your recipient opens the link they will be asked for the secret PIN and see your secret message.
 The PIN is (typically) numeric and easy to pass by a voice call or text message.
 Each link can be opened only **once** and the number of attempts to enter a wrong PIN is limited to 3 times by default.

[![ScreenShot](https://raw.github.com/umputun/secrets/master/webapp/demo/demo.png)](https://www.youtube.com/watch?v=erT1QYiDD-Q)

## How safe is this thing

- It doesn't keep your original message or PIN anywhere, but encrypts your message with PIN (hashed as well)
- It doesn't keep any sensitive info in any logs
- It doesn't keep anything on a disk in any form (in case of default engine)
- As soon as a message is read or expired it will be deleted and destroyed permanently
- In order to steal your message, bad guys would need access to your link as well as PIN code

_Feel free to suggest any other ways to make the process safer._

## Installation

1. Download `docker-compose.yml` and `secrets-nginx.conf`
1. Adjust your local `docker-compose.yml` with:
    - TZ - your local time zone
    - SIGN_KEY - something long and random
    - MAX_EXPIRE - maximum lifetime period, default 24h
    - PIN_SIZE - size (in characters) of the pin, default 5
    - PIN_ATTEMPTS - maximum number of failed attempts to enter pin, default 3
1. Setup SSL:
    - The system can make valid certificates for you automatically with integrated [nginx-le](https://github.com/umputun/nginx-le). Just set:
        - LETSENCRYPT=true
        - LE_EMAIL=name@example.com
        - LE_FQDN=www.example.com
    - In case you have your own certificates, copy them to `etc/ssl` and set:
        - SSL_CERT - SSL certificate (file name, not path)
        - SSL_KEY - SSL key (file name, not path)
1. Run the system with `docker-compose up -d`. This will download a prepared image from docker hub and start all components.
1. if you want to build it from sources - `docker-compose build` will do it, and then `docker-compose up -d`.

_See [docker-compose.yml](https://github.com/umputun/secrets/blob/master/docker-compose.yml) for more details_

### Technical details

**Safesecret** usually deployed via docker-compose and has two containers in:

- application `secrets` container providing both backend (API) and frontend (UI)
- nginx-le container with nginx proxy and let's encrypt SSL support

Application container is fully functional without nginx proxy and can be used in stand-alone mode. You may want such setup
in case you run **safesecret** behind different proxy, i.e. haproxy, AWS ELB/ALB and so on.

## API

**Safesecret** provides trivial REST to save and load messages.

### Save message

`POST /api/v1/message`, body - `{"message":"some top secret info", "exp": 120, "pin": "12345"}`

- `exp` expire in N seconds
- `pin` fixed-size pin code
    ```
        $ http POST https://safesecret.info/api/v1/message pin=12345 message=testtest-12345678 exp:=1000

        HTTP/1.1 201 Created

        {
            "exp": "2016-06-25T13:33:45.11847278-05:00",
            "key": "f1acfe04-277f-4016-518d-16c312ab84b5"
        }
    ```

### Load message

`GET /api/v1/message/:key/:pin`

    ```
        $ http GET https://safesecret.info/api/v1/message/6ceab760-3059-4a52-5670-649509b128fc/12345

        HTTP/1.1 200 OK

        {
            "key": "6ceab760-3059-4a52-5670-649509b128fc",
            "message": "testtest-12345678"
        }
    ```

### ping

`GET /api/v1/ping`

    ```
    $ http https://safesecret.info/api/v1/ping

    HTTP/1.1 200 OK

    pong
    ```

### Get params

`GET /api/v1/params`

    ```
    $ http https://safesecret.info/api/v1/params

    HTTP/1.1 200 OK

    {
        "max_exp_sec": 86400,
        "max_pin_attempts": 3,
        "pin_size": 5
    }
    ```
