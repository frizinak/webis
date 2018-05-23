# API

## Set key O(1)

`POST /set`

Headers:
```
X-Key: <key>
X-Namespace: <namespace>
X-Tags: <tag-1>
X-Tags: <tag-2>
X-Tags: <tag-N>
X-TTL: <ttl-in-seconds>
```

Body: `<value>`

## Get key O(1)

`GET /get`

Headers:
```
X-Key: <key>
X-Namespace: <namespace>
```

## Get tag keys O(n)

`GET /get`

Headers:
```
X-Tags: <tag>
X-Namespace: <namespace>
```

## Delete by key O(1)

`POST /del`

Headers:
```
X-Key: <key>
X-Namespace: <namespace>
```

## Delete by tag O(n)

`POST /del`

Headers:
```
X-Tags: <tags>
X-Namespace: <namespace>
```

## List keys O(n)

`GET /list`

Headers:
```
X-Key: <key-with-*-wildcard-expansion>
X-Namespace: <namespace>
```

## List tags O(n)

`GET /list`

Headers:
```
X-Tags: <tag-with-*-wildcard-expansion>
X-Namespace: <namespace>
```

## Purge namespace O(n)

`POST /purge`

Headers:
```
X-Namespace: <namespace>
```

## Purge namespace O(n)

`POST /purge-all`

Headers: `<none>`
