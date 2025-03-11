# 本地开发环境搭建

在开始开发前，你需要为整个项目安装并初始化 `pre-commit`，

``` bash
❯ pre-commit install
```

目前我们使用了四个工具: `isort`、`black`、`flake8`、`mypy`，它们能保证你的每一次提交都符合我们预定的开发规范。

## 准备 Python 开发环境

1. 安装 Python 3.8

可以使用 [pyenv](https://github.com/pyenv/pyenv) 管理本地的 python 环境

- 依照 [相关指引](https://github.com/pyenv/pyenv#getting-pyenv) 安装 pyenv

- 使用 pyenv 安装 Python 3.8

```bash
❯ pyenv install 3.8.13
```

2. 安装项目依赖

本项目使用 [poetry](https://python-poetry.org/) 管理项目依赖。

- 安装 poetry

```bash
❯ pip install poetry==1.3.2
```

- 使用 poetry 安装依赖

```bash

❯ poetry install --no-root
```

完成依赖安装后，便可以使用 poetry 启动项目了，常用命令：
- poetry shell：进入当前的 virtualenv
- poetry run {COMMAND}：使用 virtualenv 执行命令
