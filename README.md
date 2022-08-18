# Blog

Web service with CRUD functionality 

## Quick Start

0. Clone the repo

```
git clone git@github.com:svileex/blog.git
cd blog
```

1. Build image

```
docker compose build
```

2. Deploy image

```
docker compose up
```

3. Test

```
curl --location --request POST 'localhost:8081/api/v1/register' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "aboba",
    "password": "test"
}'
```

## API

[microblog.yaml](./microblog.yaml)
