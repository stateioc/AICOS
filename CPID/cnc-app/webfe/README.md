蓝鲸应用前端开发框架使用指南
------------------------

它是基于 `Vue.js` 研发的蓝鲸体系前端分离工程的单页面应用模板（BKUI-CLI），包括了：

- 基础工程化能力，开箱即用，无需过多配置，开发完成直接在蓝鲸PaaS可部署
- 基础 mock 服务，帮助开发者快速伪造接口数据，测试前端
- 蓝鲸前端/设计规范，提供统一设计及代码检测
- bk-magic-vue 组件库，提供丰富的组件
- 蓝鲸前端通用逻辑，包含登录模块、异步请求管理等
- 最佳实践以及开发示例



# 本地开发

#### 安装依赖包
```
npm install
```

#### 配置host
配置指定域名
```
127.0.0.1 域名
```

#### 检查配置文件

注意：运行之前，请检查配置文件**.babelrc**、**eslintrc.js**等以.开头的配置文件是否存在。部分操作系统会默认隐藏这类文件，导致在推送到代码仓库时漏掉，最终影响部署结果。

#### 启动服务
```
npm run dev
```

#### 打开链接

> 开发域名及端口的配置都可在`bk.config.js`中修改

# 前后端分离
当前代码仅仅是应用前端，作为前后端分离架构，还需要后端服务，前后端以ajax+json进行数据处理，因此，需要

#### 新建后端服务模块（开发者中心）

#### 配置APP ID
- 开发环境下编辑根目录下`env.js`，修改`development.BKPAAS_APP_ID`，用于本地开发，线上部署时会自动注入

#### 配置后端接口
- 本地开发修改`env.js`中`development`的`AJAX_URL_PREFIX`字段，作为接口的url前缀
- 线上部署测试环境修改`env.js`中`stag`方法里的的`AJAX_URL_PREFIX`字段，作为接口的url前缀
- 线上部署测正式境修改`env.js`中`production`方法里的的`AJAX_URL_PREFIX`字段，作为接口的url前缀

#### 配置用户登录态信息接口，作为前端判断登录状态的验证
> 打开首页，前端会以/user 来发起用户信息请求，如果没登录会重定向回登录页面
> 整个框架自带登录实现，在刚打开时，如果没有登录会直接跳到登录页，如果打开后，登录过期（接口返回401状态）会弹出登录窗口
###### 本地开发环境登录配置
####### 用户登录配置
`main.js` 入口会首先进行用户鉴权，如果没有用户信息，会跳转至登录页面。
####### 登录地址配置
本地开发需要修改根目录下`env.js` 配置  `development.BK_LOGIN_URL` 与 `development.BKPAAS_APP_ID`，线上部署会自动注入
####### 登录鉴权
本地登录，会在本地服务 `pass-server/middleware/user.js` 转发登录接口请求，登录接口在 `env.js` 里面的 `development.BK_LOGIN_URL` 注入

需要后端接口在 `/user` 路径下实现获取当前登录用户的接口, 接口规范如下
```json
{
    "code": 0,
    "data": {
        "bk_username": "test",
        "avatar_url": ""
    },
    "message": "用户信息获取成功"
}
```

#### 后端服务需要解决跨域问题，推荐使用CORS方案
详情查看MDN https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Access_control_CORS

# 打包构建（生成dist目录）
```
npm run build
```

## 类型项目目录结构

使用 `bkui-cli` 初始化后，项目目录结构如下：

```bash
|-- ROOT/               # 项目根目录
    |-- .babelrc        # babel 配置
    |-- .eslintignore   # eslintignore 配置
    |-- .eslintrc.js    # eslint 配置
    |-- .browserslistrc      # 编译代码兼容浏览器配置
    |-- env.js      # 项目启动配置的环境变量等
    |-- .bk.config.js      # 🌟项目编译配置文件 提供 devServe & mockServer &生产包功能集合
    |-- .gitignore      # gitignore 配置
    |-- README.md       # 工程的 README
    |-- index-dev.html  # 本地开发使用的 html
    |-- index.html      # 构建部署使用的 html
    |-- package-lock.json # package-lock file
    |-- package.json    # package.json，我们在提供了基本的 doc, dev, build 等 scripts，详细内容请参见文件
    |-- postcss.config.js # postcss 配置文件，我们提供了一些常用的 postcss 插件，详细内容请参见文件
    |-- paas-server/            # 🌟可部署运行编译后代码的node服务
    |   |-- middleware  # 公共的中间件
    |       |-- user.js    # 登录中间件
    |   |-- index.js    # 启动服务入口
    |-- src/            # 🌟实际项目的源码目录
    |   |-- App.vue     # App 组件
    |   |-- main.js     # 主入口
    |   |-- public-path.js  # __webpack_public_path__ 设置
    |   |-- api/        # 对 axios 封装的目录
    |   |   |-- cached-promise.js # promise 缓存
    |   |   |-- index.js          # axios 封装
    |   |   |-- request-queue.js  # 请求队列
    |   |-- common/     # 项目中常用模块的目录
    |   |   |-- auth.js     # auth
    |   |   |-- bkmagic.js  # bk-magic-vue 组件的引入
    |   |   |-- bus.js      # 全局的 event bus
    |   |   |-- demand-import.js    # 按需引入 bk-magic-vue 的组件
    |   |   |-- fully-import.js     # 全量引入 bk-magic-vue 的组件
    |   |   |-- preload.js  # 页面公共请求即每次切换 router 时都必须要发送的请求
    |   |   |-- util.js     # 项目中的常用方法
    |   |-- components/     # 项目中组件的存放目录
    |   |   |-- auth/       # auth 组件
    |   |   |   |-- index.css   # auth 组件的样式
    |   |   |   |-- index.vue   # auth 组件
    |   |   |-- exception/      # exception 组件
    |   |       |-- index.vue   # exception 组件
    |   |-- css/            # 项目中通用的 css 的存放目录。各个组件的样式通常在组件各自的目录里。
    |   |   |-- app.css     # App.vue 使用的样式
    |   |   |-- reset.css   # 全局 reset 样式
    |   |   |-- variable.css    # 存放 css 变量的样式
    |   |   |-- mixins/     # mixins 存放目录
    |   |       |-- scroll.css  # scroll mixin
    |   |-- images/         # 项目中使用的图片存放目录
    |   |   |-- 403.png     # 403 错误的图片
    |   |   |-- 404.png     # 404 错误的图片
    |   |   |-- 500.png     # 500 错误的图片
    |   |   |-- building.png # 正在建设中的图片
    |   |-- router/         # 项目 router 存放目录
    |   |   |-- index.js    # index router
    |   |-- store/          # 项目 store 存放目录
    |   |   |-- index.js    # store 主模块
    |   |   |-- modules/    # 其他 store 模块存放目录
    |   |       |-- example.js  # example store 模块
    |   |-- views/          # 项目页面组件存放目录
    |       |-- 404.vue     # 404 页面组件
    |       |-- index.vue   # 主入口页面组件，我们在这里多使用了一层 router-view 来承载，方便之后的扩展
    |       |-- example1/   # example1 页面组件存放目录
    |       |   |-- index.css   # example1 页面组件样式
    |       |   |-- index.vue   # example1 页面组件
    |       |-- example2/   # example2 页面组件
    |       |   |-- index.css   # example2 页面组件样式
    |       |   |-- index.vue   # example2 页面组件
    |-- static/             # 静态资源存放目录，通常情况下， 这个目录不会人为去改变
    |   |-- lib-manifest.json   # webpack dll 插件生成的文件，运行 npm run dll 或者 npm run build 会自动生成
    |   |-- lib.bundle.js       # webpack dll 插件生成的文件，运行 npm run dll 或者 npm run build 会自动生成
    |   |-- images/         # 图片静态资源存放目录
    |       |-- favicon.ico # 网站 favicon
```
