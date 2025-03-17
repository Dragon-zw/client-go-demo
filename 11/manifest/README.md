# 1 部署

## 1.1 构建 Go 镜像

定义 Dockerfile 文件：

```dockerfile
FROM docker.m.daocloud.io/library/golang:1.23.7 as builder
# 后期需要优化指定运行的用户和用户组
WORKDIR /app

COPY . .
# 构建
RUN CGO_ENABLED=0 go build -o ingress-manager main.go

FROM docker.m.daocloud.io/library/alpine:3.15.3

WORKDIR /app

COPY --from=builder /app/ingress-manager .

CMD ["./ingress-manager"]
# docker build -t dragonzw/ingreminikube mount 11:/home/docker/client-goss-manager:1.0.0 .
```

将 MiniKube 的目录进行挂载到宿主机当中

```shell
$ minikube ssh
# 其路径为 /home/docker/client-go
$ mkdir -p client-go && cd client-go

# 进行 Code 代码的挂载
$ minikube mount 11:/home/docker/client-go
📁  将主机路径 11 挂载到虚拟机中作为 /home/docker/client-go ...
    ▪ 挂载类型： 9p
    ▪ 用户 ID：      docker
    ▪ 组 ID：docker
    ▪ 版本：      9p2000.L
    ▪ 消息大小：262144
    ▪ 选项：map[]
    ▪ 绑定地址：10.211.55.2:54217
🚀  用户空间文件服务器ufs starting
✅  成功将 11 挂载到 /home/docker/client-go

📌  注意：此进程必须保持活动状态才能访问安装......
```

这样就可以查看到代码的目录信息：

![img](https://cdn.nlark.com/yuque/0/2025/png/2555283/1741751588293-82be9d13-20ec-4d52-802c-b68ff1f07d56.png)

然后进行 Dockerfile 构建镜像

```shell
$ docker build -t dragonzw/ingress-manager:1.0.0 .

# 可以使用该镜像进行直接使用(推荐方式)
$ docker pull wangtaotao2015/ingress-manager:1.0.0
$ docker pull dragonzw/ingress-manager:1.0.0 
# $ docker pull docker.m.daocloud.io/wangtaotao2015/ingress-manager:1.0.0
```

![img](https://cdn.nlark.com/yuque/0/2025/png/2555283/1741752366935-b9a21ae7-0a8c-46cd-9a4d-541f057fefbd.png)

可以选择导入镜像：

```shell
$ docker load -i ingress-manager-1.0.0.tar 
a1c01e366b99: Loading layer [==================================================>]  5.855MB/5.855MB
c07b288c0329: Loading layer [==================================================>]  2.048kB/2.048kB
a60c56e9a23b: Loading layer [==================================================>]  46.79MB/46.79MB
Loaded image: wangtaotao2015/ingress-manager:1.0.0
```

Reference：[X86 Docker镜像转换为 ARM 架构镜像](https://blog.csdn.net/m0_56659620/article/details/143176075) 需要注意镜像的 OS 架构问题！

## 1.2 部署 Deployment

```bash
# 创建 Deployment 控制器
$ kubectl create deployment ingress-manager --image=dragonzw/ingress-manager:1.0.0 --dry-run=client -o yaml
# ingress-manager-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: ingress-manager
  name: ingress-manager
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ingress-manager
  template:
    metadata:
      labels:
        app: ingress-manager
    spec:
      # 需要关联 ServiceAccount
      serviceAccountName: ingress-manager-sa
      containers:
        - image: dragonzw/ingress-manager:1.0.0
          name: ingress-manager
          resources: {}
$ kubectl create -f manifest/ingress-manager-deployment.yaml
deployment.apps/ingress-manager created
```

## 1.3 创建 ServiceAccount

```bash
# 创建 ServiceAccount 服务账号
$ kubectl create serviceaccount ingress-manager-sa --dry-run=client -o yaml
# ingress-manager-sa.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ingress-manager-sa
  namespace: default
$ kubectl create -f manifest/ingress-manager-sa.yaml
serviceaccount/ingress-manager-sa created
```

## 1.4 创建 RBAC

### 1.4.1 创建 ClusterRole
之前使用的是 Role，命名空间级别资源（对于 Operator 来说权限范围过小，需要使用集群级别资源）
```bash
# 创建 RBAC 权限
$ kubectl create role ingress-manager-role \
  --resource=ingress,service \
  --verb list,watch,create,update,delete --dry-run=client -o yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  creationTimestamp: null
  name: ingress-manager-role
rules:
- apiGroups:
  - ""
  resources:
  - services
  verbs:
  - list
  - watch
  - create
  - update
  - delete
- apiGroups:
  - networking.k8s.io
  resources:
  - ingresses
  verbs:
  - list
  - watch
  - create
  - update
  - delete
```

![img](https://cdn.nlark.com/yuque/0/2025/png/2555283/1741758435734-a585c56e-10ff-4c01-b3b3-6a71a586fb4f.png)

```yaml
# ingress-manager-clusterrole.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ingress-manager-role
  namespace: default
rules:
  - apiGroups:
      - ""
    # 将 Service 的权限进行回收
    resources:
      - services
    verbs:
      - list
      - watch
  - apiGroups:
      - networking.k8s.io
    resources:
      - ingresses
    verbs:
      - list
      - watch
      - create
      - update
      - delete
# 引用资源对象文件
$ kubectl create -f manifest/ingress-manager-role.yaml
role.rbac.authorization.k8s.io/ingress-manager-role created
```

### 1.4.2 创建 ClusterRoleBinding

之前使用的是 RoleBinding，命名空间级别资源（对于 Operator 来说权限范围过小，需要使用集群级别资源）。然后进行 ClusterRoleBinding 绑定：

```bash
$ kubectl create rolebinding ingress-manager-rolebinding \
  --role=ingress-manager-role \
  --serviceaccount=default:ingress-manager-sa --dry-run=client -o yaml
# ingress-manager-clusterrolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ingress-manager-rolebinding
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ingress-manager-role
subjects:
  - kind: ServiceAccount
    name: ingress-manager-sa
    namespace: default
# 引用资源对象文件
$ kubectl create -f manifest/ingress-manager-rolebinding.yaml
rolebinding.rbac.authorization.k8s.io/ingress-manager-rolebinding created
```

## 1.5 重新部署 Deployment

```shell
$ kubectl apply -f manifest/ingress-manager-deployment.yaml
deployment.apps/ingress-manager change

# Pod Log 日志会查看 Cluster Scope 的权限
# 这边开发的 Operator 是监听所有的 Namespace 下的 Ingress，但是使用的 ServiceAccount 只有 default namespace 下的权限
# 所以 Operator 程序报错了。
```

重新修改 Role 修改为 ClusterRole，已经 RoleBinding 修改为 ClusterRoleBinding。

## 1.6 测试

```shell
$ kubectl run nginx-demo --image=nginx:latest 
pod/nginx-demo created
$ kubectl expose pod nginx-demo --port=80 --target-port=80
service/nginx-demo exposed

###########################################################################
# 添加 Annotation
###########################################################################
$ kubectl edit svc nginx-demo 
apiVersion: v1
kind: Service
metadata:
  # 添加 Annotation 字段内容
  annotations:
    ingress/http: "true"
# 保存退出
service/nginx-demo edited

# 可以自动创建 Ingress 的资源对象
$ kubectl get ingress
NAME         CLASS   HOSTS         ADDRESS   PORTS   AGE
nginx-demo   nginx   example.com             80      5s
$ kubectl get ingress nginx-demo -o yaml 
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  creationTimestamp: "2025-03-12T06:50:59Z"
  generation: 1
  name: nginx-demo
  namespace: default
  ownerReferences:
  - apiVersion: v1
    blockOwnerDeletion: true
    controller: true
    kind: Service
    name: nginx-demo
    uid: 2e94c233-c18b-4510-a143-a6bb1b7784fa
  resourceVersion: "1680984"
  uid: 2bd3d763-51a4-4d9d-bf40-bbbab9c11046
spec:
  ingressClassName: nginx
  rules:
  - host: example.com
    http:
      paths:
      - backend:
          service:
            name: nginx-demo
            port:
              number: 80
        path: /
        pathType: Prefix
status:
  loadBalancer: {}

###########################################################################
# 删除 Annotation
###########################################################################
$ kubectl edit svc nginx-demo 
apiVersion: v1
kind: Service
metadata:
  # 删除 Annotation 字段内容
  annotations:
    ingress/http: "true"
# 保存退出
service/nginx-demo edited

# Ingress 会自动删除
$ kubectl get ingress
No resources found in default namespace.
```

手动通过 client-go 创建 Controller 过，才能知道 kubebuilder 的便捷和友好！

⚠️注意：

1. 开发团队对 Kubernetes 不是特别了解，使用 Deployment，Service 使用门槛比较高，需要有效的降低使用门槛
2. 是否可以将 manifest 的目录的资源清单进行优化