# Kubernetes Harbor Exporter

该 Exporter 基于用户 c4p0 开源的 Harbor Exporter 开发，删除与重写了其中相关内容，使其成为适用于镜像数量在 3000 左右也能正常工作。但是镜像数更多的场景没有信心，因为 blob 的数量数量可能会很多，这里面做的基本都是临时方案。

但是其中有很多功能是基于 k8s 的 api 实现的，这种方式也破坏该 Exporter 对于 docker compose 版 Harbor 的支持，想再回馈上游已经是不可行的了，只能成为 fork 出的新分支。

## 总体流程

原版程序通过 Prometheus 官方提供的 client-golang 与 Prometheus 进行通信，用户需要在提供`Describe`和`Collect`函数去采集各个指标。并启动 http 服务器，通过`/metric`为 Prometheus 暴露指标。

源程序中全部指标均为通过 harbor 的 http api 取得，在部分指标上存在版本不兼容的问题，如项目配额相关指标。部分指标存在不可用问题，如采集 repo 项目信息时，源项目需构建共计 600 个 http 请求，耗时 50 多秒，超出了允许的最大超时时间 15 秒。以及磁盘信息采集不适用 nfs 卷等。

所以将 repo 相关数据的获取改为了直接通过 pg 去获取的方式，pg 相关的信息通过 kube api 去获取，需要为 pod 设置 serviceaccount。经测试，正常使用的情况下不会对 pq 造成压力，但是如果 pod 被意外的关闭，可能会导致残留两个 idle 的 pg 连接。Harbor database 的默认最大连接数是 100，harbor 相关组件会使用其中 30 个左右的空闲连接。也就是说如果这个 exporter crash backoff 最多 35 次，就会导致下一次 harbor 相关组件无法正常重启。不知道 harbor database 组件里面有没有定时清理 idle 连接的定时任务，看了下系统设置好像没设置，这里可能需要多关注。

比较理想的聚合数据获取方法还是应该单独建立字段去维护，当前 repo 表中的 pull_count 就是这样维护的。可能是怕修改频繁带来死锁问题，每次用数据库统计又会带来性能问题，所以官方还没有提供相关集成的 exporter 方案。

## 详细流程

- harbor_up

  源项目就有，没动过，全部 collection 函数允许完成后返回 1

- harbor_project_count_total、harbor_repo_count_total、harbor_replication_tasks、harbor_replication_status

  源项目就有，没动过，通过 harbor 提供的 api 接口去采集数据，请求数量极少，响应速度快

- harbor_system_volumes_bytes

  通过 kubeapi 执行 pod/exec 请求运行`sh -c df e.opts.storage`得到。e.opts.storage 是 configmap 中 registry 的 config.yml 提供的。该方式仅适用于通过 filesystem 挂载的存储。

- harbor_repositories_pull_total，harbor_repositories_push_total

  通过 sql 得到，相关语句如下，执行一次

  ```sql
  SELECT
    r.repository_id as repo_id,
    r.name as repo_name,
    r.pull_count as pull_count,
    count(al.log_id) as push_count
  FROM
    repository as r
    JOIN access_log as al ON al.repo_name = r.name
  WHERE
    al.operation = 'push'
  GROUP By
    r.repository_id;
  ```

- harbor_repositories_tags_total

  通过 sql 得到，相关语句如下，目前执行大概 600 次，不过没有序列化过程，还好。如果能和上一个表拼在一起就更好了，可惜不能写 views 或者存储过程去破坏数据结构。多个一对多的表 join 起来聚合函数统计会出问题也不知道咋处理，所以就只能有一个 repo 就查一次了。

  ```sql
  SELECT
  count(a.id) as tag_count
  FROM
  artifact as a
  JOIN repository as r ON a.repo = r.name
  WHERE
  r.repository_id = $1;
  ```

- harbor_image_pull_count

  通过 sql 得到，相关语句如下，执行一次

  ```sql
  SELECT
  a.repo as repo_name,
  a.tag as tag_name,
  count(al.log_id) as pull_count
  FROM
  access_log as al
  JOIN artifact as a ON a.repo = al.repo_name
  AND a.tag = repo_tag
  WHERE
  al.operation = 'pull'
  GROUP BY
  a.repo,
  a.tag;
  ```

- harbor_project_size

  通过 sql 得到，相关语句如下，执行一次

  ```sql
  SELECT
  p.name as project_name,
  sum(b.size) as size

  FROM
  blob AS b
  JOIN artifact_blob AS ab ON ab.digest_blob = b.digest
  JOIN artifact AS a ON a.digest = digest_af
  JOIN repository AS r ON a.repo = r.name
  JOIN project AS p ON r.project_id = p.project_id
  GROUP BY
  p.name;
  ```

- harbor_database_health

  直接 ping 一下地址端口，没报错就是 1

- harbor_database_connections

  通过`select count(1) from pg_stat_activity;`得到同步任务数量