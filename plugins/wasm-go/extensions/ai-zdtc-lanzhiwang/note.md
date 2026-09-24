---------------------------------------------------------------------------------------------

```bash
docker run -ti --rm --name golang_1_24_1 \
-v /Users/huzhi/work/code/py_code/higress-group/higress:/higress \
-w /higress golang:1.24.1 bash

REGISTRY=registry-dx.wair.ac.cn/taichu-studio/ PLUGIN_NAME=ai-zdtc-lanzhiwang make build-image -n --dry-run


```

---------------------------------------------------------------------------------------------


