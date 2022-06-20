# docker 部署并运行普罗米修斯

**docker run ：**创建一个新的容器并运行一个命令；*命令*运行时会在本地寻找镜像，找不到的时候就*会去Docker* Hub上面搜索并*下载*后运行

1、执行命令安装:

```
$ docker run --name prometheus -d -p 127.0.0.1:9090:9090 quay.io/prometheus/prometheus
```

2、安装grafana

```
brew update
brew install grafana
```

当安装成功后，你可以使用默认配置启动程序

```
grafana-server -homepath /usr/local/Cellar/grafana/8.3.6/share/grafana/
```

此时，你可以打开页面 `http://localhost:3000`， 访问 Grafana 的 web 界面。

3、grafana设置prometheus数据源

使用默认账号 admin/admin 登录 grafana

在 Dashboard 首页，点击添加数据源

配置 Prometheus 数据源

4、修改promethues配置，设置拉取目标地址

vi /etc/prometheus/prometheus.yml

配置前先ifconfig看一下本机ip，因为docker内使用localhost无法获取宿主ip

添加配置项

```
- job_name: client
  honor_timestamps: true
  scrape_interval: 15s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  follow_redirects: true
  static_configs:
  - targets:
    - 192.168.1.133:27000
```

重启docker即可在localhost:9090上看到数据(选择一下expression)

5、grafana 配置一下dashboard的query，选择对应的expresion即可展示数据

