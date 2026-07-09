https://10.122.1.120/vminstance
admin
admin@123

ssh root@172.16.10.98
67R5^WSCyz9uY+
/usr/local/bin/sshpass -p '67R5^WSCyz9uY+' ssh root@172.16.10.98

higress
http://172.16.10.98:32333
admin
admin

```bash
##############################################################
docker run -ti --rm --name golang_1_24_1 \
-v /Users/huzhi/work/code/py_code/higress:/higress \
-w /higress golang:1.24.1 bash

##############################################################
docker run -ti --rm \
--entrypoint /usr/bin/env \
-v /Users/huzhi/work/code/py_code/higress:/higress \
-p 0.0.0.0:2022:22 \
--name golang-ssh-server \
lanzhiwang/golang:sha-c65f57e bash

sudo /usr/sbin/sshd

docker run -ti --rm \
--entrypoint /usr/bin/env \
--name golang-ssh-client \
lanzhiwang/golang:sha-c65f57e bash

##############################################################

docker run -d \
-v /Users/huzhi/work/code/py_code/higress:/home/lanzhiwang/higress \
-p 0.0.0.0:2022:22 \
--name golang-ssh-server \
lanzhiwang/golang:sha-c65f57e

ssh-keygen -t rsa
ssh-copy-id -i ~/.ssh/docker.pub -p 2022 lanzhiwang@127.0.0.1
ssh -p 2022 lanzhiwang@127.0.0.1 -i ~/.ssh/docker

```
