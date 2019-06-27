**树莓派上的Edgexfoundry实战**

---

此文档用于详细描述边缘计算框架degexfoundty部署到树莓派上的方法，可能出现的bug以及必要的说明

**目录**

[TOC]


# 1 Edgexfoundry框架

## 1.1 简介
+ 官方文档 ：<https://docs.edgexfoundry.org>
  + Core Service层：**core-data**, **core-metadata**, **core-command**
  + Support Service层
  + Export Service层
  + Device Service层：**device-mqtt**
  + EdgeX UI：<https://github.com/edgexfoundry/edgex-ui-go>
+ 官方github： <https://github.com/edgexfoundry>
+ 官方维基：<https://wiki.edgexfoundry.org>

## 1.2 device-opcua微服务

device-opcua微服务位于Device Service层，与基于OPCUA协议的设备通讯

### 1.2.1 目标

针对基于**OPC-UA协议**的设备/传感器，基于官方github给出的 [device-skd-go](<https://github.com/edgexfoundry/device-sdk-go>)的delhi分支下的模板，定制**device-opcua微服务**的**golang**版本，调用opcua的 [go SDK](<https://github.com/gopcua/opcua>)接口,可以实现对此类设备的**注册**、**管理**、**控制**等，制作此微服务的**Docker镜像**，并部署在**树莓派3b+**，实现对OPCUA服务端节点的**读取、设置、监听**等操作。

### 1.2.2 准备

可提前准备好配置文件和Device Profile，方便设备微服务读取。服务启动后，配置信息可以通过[consul](<https://www.consul.io/>)服务注入，Device Profile, Device的信息等可利用[Postman](<https://www.getpostman.com/>)软件调用core-matadata服务的[API](<https://docs.edgexfoundry.org/core-metadata.html>)添加

1. `configuration.toml`文件提供device-opcua服务的信息、consul服务的信息、其他需要和设备服务交互的微服务的信息、Device信息（包含**Device Profile的目录**）、日志信息、预定义Schedule和SchedukeEvent信息（包含要**定时执行的命令**）、预定义设备信息（包含**设备的Addressable信息**）。 

2. `configuration-driver.toml`文件提供OPCUA Server的NodeID与deviceResource的对应关系，以及监听操作的端点信息和设备资源对应关系

3.  `OpcuaServer.yaml`作为设备的Device Profile, 有关它的书写参见[相关资料](#相关资料)1 2
   

*注* ：configuration.toml, configuration-driver.toml和Device Profile应确定好**唯一**的映射关系

### 1.2.4 已实现功能
  
  1. OPCUA设备管理
  2. 监听OPCUA节点上的值
  3. 对指定节点进行读取操作
  4. 对指定节点进行设置操作
  5. 根据预定义的schedule执行命令

Device Service的编写参考官方文档：<https://docs.edgexfoundry.org/Ch-GettingStartedSDK-Go.html>

代码已提交至github仓库：<https://github.com/Burning1020/device-opcua-go>

### 1.2.5 包管理

包管理工具可以自动下载代码所需要的依赖，不需要手工添加，只需要指定好包的地址和版本

+ edgex-go的deihi分支采用第三方包管理工具[glide](<https://glide.sh>)，利用glide.yaml下载代码所需依赖

+ go 1.11版本支持“modules”特性，使用[vgo](<https://godoc.org/golang.org/x/vgo>)作为包管理工具，利用go.mod文件下载配置依赖，master分支采用vgo

*bug提示* ：下载golang.org的包会被墙，要修改go.mod用github.com/golang替代源

### 1.2.6 Docker镜像制作

在`$GOPATH/src/github.com/edgexfoundry/device-opcua-go`目录下：

+ 执行`make build`命令编译可执行二进制文件device-opcua-go至cmd目录
+ 执行`make run`命令运行此可执行文件
+ 执行`make build-arm64`命令构建镜像，此镜像可放在树莓派内的Docker容器中运行，配置文件的读取参见[1.2.4](#1.2.4代码)

## 1.3 Export微服务

Export 微服务可将数据导出到西门子工业云平台MindSphere

先将Open Edge Diver Kit置为export-distro的客户端，然后对其进行初始化并采集数据上云所需参数，最后发送数据

详见: Export-go: https://github.com/Burning1020/export-go

# 2 OPC统一架构

+ OPC中国：<http://opcfoundation.cn/developer-tools/specifications-unified-architecture/index.aspx>

+ OPCUA虚拟设备：<https://www.prosysopc.com/products/opc-ua-simulation-server>


# 3 树莓派

## 3.1 准备

详见[相关资料](#相关资料)3

+ 烧录系统：推荐ubuntu 16.04 链接: https://pan.baidu.com/s/1gRXex4njLKi6dAHcqzrgjw 提取码: 9y5h

+ 安装Docker：<https://docs.docker.com/install>

+ 安装Docker-Compose：<https://docs.docker.com/compose/install>

  *提示：*安装出现问题参见[相关资料](#相关资料)3

## 3.2 树莓派上部署egdexfoundry服务

修改docker-compose.yml文件，按需修改服务的image地址为`nexus.edgexfoundry.org:10004/<微服务>-go-arm64:<tag号>`，具体详见：<https://docs.edgexfoundry.org/Ch-GettingStartedUsersNexus.html>

拉取镜像的快慢取决于网络环境，网络环境差的可以参考[相关资料](#相关资料)3

## 3.3 树莓派上部署device-opcua服务

### 3.3.1 从docker hub上获取

将构建和的镜像打标签后推送至docker hub


```bash
docker tag
  
docker push
  ```


### 3.3.2 从私有仓库中获取

详见：[同一局域网下搭建私有Docker仓库](<https://my.oschina.net/u/3746745/blog/1811532>)

同一局域网下，服务器和树莓派IP地址已知，让树莓派中运行的Docker拉取服务器中的镜像，在服务器中执行以下命令：

1. 创建本地仓库：`docker pull registry`

2. 容器中启动仓库： `docker run -d –p <端口:端口> <仓库名>`

3. 查看本地仓库：`curl 127.0.0.1:<端口>/v2/_catalog`

4. 打标签：`docker tag <目标镜像>  <服务器IP:端口/镜像名>`

5. 修改push的HTTPS要求: `vim /etc/docker/daemon.json`            

   { "insecure-registries": ["<服务器IP:端口>"] }

6. 重启docker：`systemctl restart docker`

7. 启动仓库，推送镜像：docker push <tag后的镜像名>

### 3.3.3 启动镜像

1. 向docker-compose.yml文件添加device-opcua服务，image地址为 <tag后的镜像名>

2. 修改pull的HTTPS要求: `vim /etc/docker/daemon.json`

   { "insecure-registries": ["<服务器IP:端口>"] }

3. 重启docker：`systemctl restart docker`

4. 拉取并启动device-opcua服务：`docker-compose up -d device-opcua`

# 4 云边协同

## 4.1 报警服务

待补充

## 4.2 Kubernetes

利用kubernetes将报警服务分发到节点并启动，当条件满足时触发报警动作

# 相关资料

1. EdgeX Tech Talks (YouTube 频道需要翻墙) ：<https://www.youtube.com/playlist?list=PLih1NL_0jlJNyzs-y_kUuni4yr8J04Nnp>

2. 边缘计算论坛：<http://www.discuz.edgexfoundry.net>

3. EdgeX Foundry中国社区：<http://www.edgexfoundry.club>

4. CSDN博客：

   + <https://blog.csdn.net/winfanchen/article/category/7702798>

   + <https://blog.csdn.net/keyoushide/article/category/8363288>

5. 阿里云云栖社区：<https://yq.aliyun.com/users/pxmocwktpc7be/own?spm=a2c4e.11153940.blogrightarea608710.4.5e2d42a4Ytd3Zc>

# 捐赠
如果您喜欢这个项目，请在github上为我点赞，您的鼓励是我最大的动力！