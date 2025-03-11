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

### 3. 请求算力资源列表

按条件筛选满足的算力可用区，行业信息

#### 请求 Request

- 请求方法 Method: **POST**
- 请求路径 URL:  ```/api/v1/query_computing_resources/```
- 请求体 Body:

```json
{
  "source": "ctyun",
  "session_identifier": "",
  "data": {
    "userID": "xxx",
    ...
  }
}
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段                 | 类型     | 位置   | 必须 | 描述     |
|--------------------|--------|------|----|--------|
| source             | string | body | 否  | 请求来源   |
| session_identifier | string | body | 否  | 用户id或者任务标识   |
| data               | object | body | 否  | 资源需求数据 |

#### 响应 Response

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
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
{
  "source": "ctyun",
  "session_identifier": "",
  "data": {
    "userID": "xxx",
    ...
  }
}
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段                 | 类型     | 位置   | 必须 | 描述     |
|--------------------|--------|------|----|--------|
| source             | string | body | 否  | 请求来源   |
| session_identifier | string | body | 否  | 用户id或者任务标识   |
| data               | object | body | 否  | 任务模版信息 |

#### 响应 Response

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

### 5. 接收用户的路径选择

#### 请求 Request

- 请求方法 Method: **POST**
- 请求路径 URL:  ```/api/v1/task_path/```
- 请求体 Body:

```json
{
  "source": "ctyun",
  "session_identifier": "",
  "data": {
    "userID": "xxx",
    ...
  }
}
```

##### 请求头 Headers

| 字段    | 类型     | 必须 | 描述        | 
|-------|--------|----|-----------|
| token | string | 是  | 请求认证token |
| Content-Type | string | 是  | application/json |

##### 请求参数 Parameters

| 字段                 | 类型     | 位置   | 必须 | 描述     |
|--------------------|--------|------|----|--------|
| source             | string | body | 否  | 请求来源   |
| session_identifier | string | body | 否  | 用户id或者任务标识   |
| data               | object | body | 否  | 执行结果数据 |

#### 响应 Response

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```