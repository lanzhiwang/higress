---------------------------------------------------------------------------------------------

```bash
docker run -ti --rm --name golang_1_24_1 \
-v /Users/huzhi/work/code/py_code/higress-group/higress:/higress \
-w /higress golang:1.24.1 bash

REGISTRY=registry-dx.wair.ac.cn/taichu-studio/ PLUGIN_NAME=ai-zdtc-lanzhiwang make build-image -n --dry-run


curl --location 'http://172.16.10.55/maas-mg/v1/chat/completions' \
--header "Authorization: Bearer iabjwhwdaryy32nyux6bmv0h" \
--header 'Content-Type: application/json' \
--data '{
  "model": "sy-dp4.1-flash",
  "messages": [
      {
          "role": "user",
          "content": "什么是redis?"
      }
  ],
  "stream": true
}'

curl --location 'http://172.16.10.55/maas-mg/v1/chat/completions' \
--header "Authorization: Bearer iabjwhwdaryy32nyux6bmv0h" \
--header 'Content-Type: application/json' \
--data '{
  "model": "sy-dp4.1-flash",
  "messages": [
      {
          "role": "user",
          "content": "什么是mysql?"
      }
  ],
  "stream": false
}'

```

---------------------------------------------------------------------------------------------


