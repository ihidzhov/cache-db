# HTTP(s) Cache Database Server

Simple in memory cache server like Memcache written in Go.

## Description

In memory server to store data in palin text format, with key and values, TTL option and some statistics.
It can br used as just cache server, or in memory database.
Data can be retreived as plain/text or application/json response.

## Getting Started

### Dependencies

* Go
* Git

### Install and run

```
git clone https://github.com/ihidzhov/cache-db.git cache-server
cd cache-server
go run .
```

## All API end points
```
POST   /set                        form-data: key, value, ttl
GET    /get?key=key
GET    /get?key=key?output=json
DELETE /delete                     form-data: key
GET    /stats
PUT    /increment                  form-data: key
PUT    /decrement                  form-data: key
```

## How it works

* To store data just send POST request to /set with form params key, value, ttl
```
curl --location 'http://localhost:8080/set' \
--form 'key="key"' \
--form 'value="value"' \
--form 'ttl="300"'
```

* To get data send GET request to /get
```
curl --location 'http://localhost:8080/get?key=key'
```

* To get data as JSON response send GET request to /get?output=json
```
curl --location 'http://localhost:8080/get?key=key&output=json'
```

## Help

Please use issues in this repo for any questions, bugs, features suggestions and so on.

## Version History

* 0.2
    * Various bug fixes and optimizations
    * See [commit change]() or See [release history]()
* 0.1
    * Initial Release

## License

This project is licensed under the MIT License - see the LICENSE.md file for details

## Acknowledgments
 
