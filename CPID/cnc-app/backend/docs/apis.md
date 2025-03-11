## 互通平台 API 文档

版本: 0.0.1

### 0. API调用协议

- 以下API在调用时需要http头中传入正确的token用于认证
- API Response返回的错误码 code 为 0 时请求正常，非0时请求不正常，需要查看message获取具体的错误信息

正常返回:

```json
{
    "code": 0,
    "message": "ok",
    "data": {}
}
```

错误返回:

```json
{
    "code": 500,
    "message": "Internal Server Error",
    "data": {}
}
```

### 1. 注册算力标识

接收厂商传入的统一算力标识，解析校验算力标识，转换成实际算力描述结构入库

#### 请求 Request

- 请求方法 Method: **POST**
- 请求路径 URL:  ```/api/v1/computing_ids/```
- 请求体 Body:

```json
{
  "computing_ids": [
    "1101tc200044013301502601009F1S1024N100P150"
  ]
}
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段            | 类型     | 位置    | 必须 | 描述                   |
|---------------|--------|-------|----|----------------------|
| computing_ids | array[string] | body    | 是 | 算力标识数组 |


#### 响应 Response

> Status: 200 OK

```json
{
    "code": 0,
    "message": "ok",
    "data": {}
}
```

### 2. 注册算力资源

周期性注册上报算力资源实例数据，直接入库，每个算力资源描述关联一个统一算力标识

#### 请求 Request

- 请求方法 Method: **POST**
- 请求路径 URL:  ```/api/v1/computing_resources/```
- 请求体 Body:
```json
{
  "computing_resources": [
    {
        "computing_id": "1101tc200044013301502601009F1S1024N100P150",
        "power_consumption": 0,
        "cpu_performance": 2000,
        "cpu_available": 0,
        "gpu_model": "RTX 3080",
        "gpu_performance": 18000,
        "gpu_memory": 16,
        "gpu_available": 0,
        "network_delay": "5ms",
        "network_performance": 100,
        "network_isixp": true,
        "network_ips": "192.168.1.1",
        "network_available": "",
        "network_ips_available": "192.168.1.1",
        "network_ports": "80,443",
        "price": 100
    }
  ]
}
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段            | 类型     | 位置    | 必须 | 描述                   |
|---------------|--------|-------|----|----------------------|
| computing_resources | array[object] | body | 是    | 算力资源数组 |
| computing_id        | string        | body | 是    | 算力标识   |
| power_consumption   | int           | body | 是    | 功耗  |
| cpu_performance     | int           | body | 是    | CPU性能  |
| cpu_available       | int           | body | 是    | CPU可用容量   |
| gpu_model           | string        | body | 是    | GPU型号  |
| gpu_performance     | int           | body | 是    | GPU性能  |
| gpu_memory          | int           | body | 是    | 显存     |
| gpu_available       | int           | body | 是    | GPU可用容量   |
| network_delay       | string        | body | 是    | 网络时延   |
| network_performance | int           | body | 是    | 网络性能   |
| network_isixp       | bool          | body | 是    | 是否为专网  |
| network_ips         | string        | body | 是    | IP列表   |
| network_available   | string        | body | 是    | 网络可用容量   |
| network_ips_available| string        | body | 是    | IP可用资源   |
| network_ports       | string        | body | 是    | 网络端口   |
| price               | int           | body | 否    | 单价     |

#### 响应 Response

> Status: 200 OK

```json
{
    "code": 0,
    "message": "ok",
    "data": {}
}
```

### 3. 查询算力资源列表

按条件筛选满足的算力可用区，行业信息

#### 请求 Request

- 请求方法 Method: **GET**
- 请求路径 URL:  ```/api/v1/computing_resources/```
- Path:

```
GET /api/v1/computing_resources/?page=1&page_size=10&resource_type=超算&region=北京&organization=中科曙光
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段            | 类型     | 位置    | 必须 | 描述                   |
|---------------|--------|-------|----|----------------------|
| resource_type | string | query | 否  | 资源类型                 |
| region        | string | query | 否  | 地理位置                 |
| organization  | int    | query | 否  | 算力标识中的服务厂商           |
| page_size     | int    | query | 是  | 分页参数 - 每页的数量         |
| page          | int    | query | 是  | 分页参数 - 页数，即第几页(从1开始) |

#### 响应 Response

data字段

| 字段      | 类型            | 描述       |
|---------|---------------|----------|
| count   | int           | 总数量      |
| results | array[object] | 分页查询到的结果 |

results列表元素

| 字段           | 类型     | 描述   |
|--------------|--------|------|
| organization | string | 服务厂商 |
| region       | string | 算力城市 |
| availability_zone| string | 可用区 |
| industry     | string | 行业   |

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "count": 2,
    "results": [
      {
        "organization": "中科曙光",
        "region": "北京",
        "availability_zone": "可用区一",
        "industry": "金融"
      },
      {
        "organization": "中科曙光",
        "region": "上海",
        "availability_zone": "可用区一",
        "industry": "教育"
      }
    ]
  }
}
```

### 4. 接受用户选择的任务模版返回算力资源

- 接收任务模版参数，返回任务id以及满足的算力资源实例列表
- 任务模版的信息直接入库，作为操作记录

#### 请求 Request

- 请求方法 Method: **POST**
- 请求路径 URL:  ```/api/v1/query_computing_resources_by_task_template/```
- 请求体 Body:

```json
{}  # 任务模版信息待定
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段            | 类型     | 位置    | 必须 | 描述                   |
|---------------|--------|-------|----|----------------------|

#### 响应 Response

data字段

| 字段      | 类型            | 描述       |
|---------|---------------|----------|
| task_id | int           | 任务id     |
| count   | int           | 总数量      |
| results | array[object] | 分页查询到的结果 |

results列表元素

| 字段                  | 类型      | 描述     |
|---------------------|---------|--------|
| id                  | int     | 算力资源id   |
| organization        | string     | 厂商   |
| region              | string  | 数据中心位置 |
| availability_zone   | string  | 可用区 |
| industry            | string  | 行业     |
| resource_type       | string  | 资源类型   |
| service_type        | string  | 服务类型   |
| power_consumption   | string  | 功耗     |
| cpu_performance     | int     | CPU性能  |
| cpu_available       | int     | CPU可用容量   |
| gpu_model           | string  | GPU型号  |
| gpu_performance     | int     | GPU性能  |
| gpu_memory          | int     | 显存     |
| gpu_available       | int     | GPU可用容量   |
| network_delay       | string  | 网络时延   |
| network_performance | int     | 网络性能   |
| network_isixp       | bool    | 是否为专网  |
| network_ips         | string  | IP列表   |
| network_available   | string   | 网络可用容量   |
| network_ips_available| string  | IP可用资源   |
| network_ports       | string  | 网络端口   |
| price               | int     | 单价     |


```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "task_id": 1,
    "count": 2,
    "results": [
      {
        "id": 1,
        "organization": "中科曙光",
        "region": "北京",
        "availability_zone": "可用区一",
        "industry": "金融",
        "resource_type": "超算",
        "service_type": "主机",
        "power_consumption": 0,
        "cpu_performance": 2000,
        "cpu_available": 0,
        "gpu_model": "RTX 3080",
        "gpu_performance": 18000,
        "gpu_memory": 16,
        "gpu_available": 0,
        "network_delay": "5ms",
        "network_performance": 100,
        "network_isixp": true,
        "network_ips": "192.168.1.1",
        "network_available": "",
        "network_ips_available": "192.168.1.1",
        "network_ports": "80,443",
        "price": 100
      },
      {
         "id": 2,
        "organization": "中科曙光",
        "region": "上海",
        "availability_zone": "可用区一",
        "industry": "教育",
        "resource_type": "超算",
        "service_type": "主机",
        "power_consumption": 0,
        "cpu_performance": 2000,
        "cpu_available": 0,
        "gpu_model": "RTX 3080",
        "gpu_performance": 18000,
        "gpu_memory": 16,
        "gpu_available": 0,
        "network_delay": "5ms",
        "network_performance": 100,
        "network_isixp": true,
        "network_ips": "192.168.1.1",
        "network_available": "",
        "network_ips_available": "192.168.1.1",
        "network_ports": "80,443",
        "price": 100
      }
    ]
  }
}
```

### 5. 接收用户的路径选择

- 接收上一步创建的任务id，与用户选择的算力资源实例id，以及结果，返回具体的厂商返回的算力资源地址
- 更新上一步创建的任务记录信息，写入结果

#### 请求 Request

- 请求方法 Method: **POST**
- 请求路径 URL:  ```/api/v1/task_path/```
- 请求体 Body:

```json
{
  "task_id": 1,
  "computing_resource_id": 1,
  "result": true
}
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段                    | 类型    | 位置   | 必须 | 描述         |
|-----------------------|-------|------|----|------------|
| task_id               | int   | body | 是  | 任务id       |
| computing_resource_id | int   | body | 是  | 算力资源id     |
| result                | bool  | body | 是  | true/false |

#### 响应 Response

data字段

| 字段           | 类型     | 描述      |
|--------------|--------|---------|
| computing_resource_url | string | 厂商的计算资源地址 |


```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "computing_resource_url": ""
  }
}
```
