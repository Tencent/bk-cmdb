# 通过Helm部署CMDB

## 要求
- [helm3](https://helm.sh/docs/intro/install/)
- [Kubernetes](https://kubernetes.io/docs/setup/), 可以本地测试可以用[minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)


## 准备 bk-cmdb 镜像
- 方式一：使用docker build构建镜像, 详情请参考[制作docker镜像](./build-image.md)
- 方式二：从docker hub下载 `docker pull ccr.ccs.tencentyun.com/bk.io/cmdb-standalone:v3.9.28`

## 制作 chart

```bash
创建一个空白的chart
# helm create bk-cmdb

修改对应的文件
# vim values.yaml

image:
  repository: ccr.ccs.tencentyun.com/bk.io/cmdb-standalone  (镜像仓库地址)
  pullPolicy: IfNotPresent
  # Overrides the image tag whose default is the chart appVersion.
  tag: ""


service:
  type: NodePort   <- 修改
  port: 8090  (对应cmdb_webserver端口) 


# vim Chart.yaml
appVersion: "v3.9.28"  <- (镜像版本)
```


## 打包使用

```bash
对格式进行检查
# helm lint bk-cmdb

打包
# helm package bk-cmdb

安装chart
# helm install bk-cmdb-chart bk-cmdb-0.1.0.tgz
```

## 检查服务是否正常启动
```bash
# kubectl get pods
NAME                                     READY   STATUS    RESTARTS   AGE
bk-cmdb-chart-7c985f78bd-xvkb7           1/1     Running   0          13h
```

## 进入容器启动服务
```bash
进入容器
# docker exec -it 容器id /bin/bash

[root@bk-cmdb-chart-7c985f78bd-xvkb7 cmdb]# pwd
/data/cmdb

启动服务
[root@bk-cmdb-chart-7c985f78bd-xvkb7 cmdb]# ./start.sh
starting: cmdb_adminserver
starting: cmdb_apiserver
starting: cmdb_authserver
starting: cmdb_cloudserver
starting: cmdb_coreservice
starting: cmdb_datacollection
starting: cmdb_eventserver
starting: cmdb_hostserver
starting: cmdb_operationserver
starting: cmdb_procserver
starting: cmdb_taskserver
starting: cmdb_toposerver
starting: cmdb_webserver
root       8809      0  0 02:28 ?        00:00:31 ./cmdb_adminserver --addrport=10.244.0.15:60004 --logtostderr=false --log-dir=./logs --v=3 --config=configures/migrate.yaml
root       8826      0  0 02:28 ?        00:00:35 ./cmdb_apiserver --addrport=10.244.0.15:8080 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false
root       8874      0  0 02:28 ?        00:00:41 ./cmdb_cloudserver --addrport=10.244.0.15:60013 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false --enable_cryptor=false
root       8893      0  1 02:28 ?        00:01:02 ./cmdb_coreservice --addrport=10.244.0.15:50009 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181
root       9123      0  0 02:28 ?        00:00:35 ./cmdb_datacollection --addrport=10.244.0.15:60005 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false
root       9539      0  1 02:28 ?        00:01:06 ./cmdb_eventserver --addrport=10.244.0.15:60009 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false
root       9775      0  0 02:28 ?        00:00:34 ./cmdb_hostserver --addrport=10.244.0.15:60001 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false
root      10073      0  0 02:28 ?        00:00:31 ./cmdb_operationserver --addrport=10.244.0.15:60011 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false
root      10094      0  0 02:28 ?        00:00:30 ./cmdb_procserver --addrport=10.244.0.15:60003 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false
root      10116      0  0 02:28 ?        00:00:38 ./cmdb_taskserver --addrport=10.244.0.15:60012 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181
root      10140      0  0 02:28 ?        00:00:31 ./cmdb_toposerver --addrport=10.244.0.15:60002 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181 --enable-auth=false
root      10159      0  0 02:28 ?        00:00:26 ./cmdb_webserver --addrport=10.244.0.15:80 --logtostderr=false --log-dir=./logs --v=3 --regdiscv=127.0.0.1:2181
process count should be: 12 , now: 12
Not Running: cmdb_authserver
```

## 导出服务并访问CMDB
```bash
# kubectl get svc
NAME                    TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                      AGE
bk-cmdb-chart           NodePort    10.109.98.83     <none>        8090:31664/TCP                 13h
```

## 运行效果
![k8s 中运行效果](./run-in-k8s.jpg)






