# sub2api 本地/服务器运行记录

> 约定：项目根目录是 `D:\data\sub2api`。  
> Docker 镜像构建必须在项目根目录执行，不能在 `deploy` 或 `backend` 目录直接执行。

## 1. 本地开发环境启动

开发 compose 文件在 `deploy/docker-compose.dev.yml`。

```powershell
cd D:\data\sub2api\deploy
docker compose -f docker-compose.dev.yml up -d --build
```

查看状态：

```powershell
docker compose -f docker-compose.dev.yml ps
```

查看日志：

```powershell
docker compose -f docker-compose.dev.yml logs -f sub2api
```

停止：

```powershell
cd D:\data\sub2api\deploy
docker compose -f docker-compose.dev.yml down
```

重新构建启动：

```powershell
cd D:\data\sub2api\deploy
docker compose -f docker-compose.dev.yml up -d --build
```

## 2. 本地构建完整 Docker 镜像

必须在项目根目录执行，因为根目录才有 `Dockerfile`、`frontend/`、`backend/`。

```powershell
cd D:\data\sub2api
docker build -t sub2api:custom .
```

常见错误：

```text
failed to read dockerfile: open Dockerfile: no such file or directory
```

原因：你不在项目根目录，例如在 `backend` 或其他目录执行了 `docker build`。  
解决：

```powershell
cd D:\data\sub2api
docker build -t sub2api:custom .
```

如果构建时报类似：

```text
Could not load /app/frontend/src/views/admin/ProfitView.vue
```

说明路由引用了新文件，但构建上下文里没有带上。先确认文件已加入 Git 或至少存在于当前工作目录：

```powershell
cd D:\data\sub2api
git status --short frontend/src/views/admin/ProfitView.vue
git add frontend/src/views/admin/ProfitView.vue
```

注意：不要在 `D:\data\sub2api\deploy` 里执行 `git add frontend/...`，那会变成找 `deploy/frontend/...`。

## 3. 服务器部署目录

服务器目录：

```bash
cd /home/sub2api-custom
```

构建自定义镜像：

```bash
docker build -t sub2api:custom .
```

启动：

```bash
docker compose -f docker-compose.local.yml up -d
```

停止：

```bash
docker compose -f docker-compose.local.yml down
```

重启：

```bash
docker compose -f docker-compose.local.yml restart
```

查看状态：

```bash
docker compose -f docker-compose.local.yml ps
```

查看服务日志：

```bash
docker compose -f docker-compose.local.yml logs -f sub2api
```

或：

```bash
docker logs --tail=100 sub2api
```

## 4. 修改代码后的本地测试流程

推荐流程：

```powershell
cd D:\data\sub2api
git status --short
```

确认新文件没有漏掉，例如：

```powershell
git add frontend/src/views/admin/ProfitView.vue
```

然后构建镜像：

```powershell
docker build -t sub2api:custom .
```

如果使用 dev compose：

```powershell
cd D:\data\sub2api\deploy
docker compose -f docker-compose.dev.yml up -d --build
```

## 5. 数据备份

在服务器项目目录执行：

```bash
cd /home/sub2api-custom
tar --xattrs --acls -czf /home/sub2api-backup-$(date +%Y%m%d-%H%M%S).tar.gz .env data postgres_data redis_data
```

备份文件示例：

```text
/home/sub2api-backup-20260616-153000.tar.gz
```

## 6. 常用排查命令

查看容器：

```bash
docker ps
docker ps -a
```

查看镜像：

```bash
docker images | grep sub2api
```

查看最近日志：

```bash
docker logs --tail=100 sub2api
```

进入容器：

```bash
docker exec -it sub2api sh
```

检查 Postgres：

```bash
docker compose -f docker-compose.local.yml exec postgres pg_isready
```

检查 Redis：

```bash
docker compose -f docker-compose.local.yml exec redis redis-cli ping
```

## 7. 注意事项

- `docker build -t sub2api:custom .` 必须在项目根目录执行。
- `docker compose -f docker-compose.dev.yml ...` 如果在本机使用，通常在 `D:\data\sub2api\deploy` 下执行。
- 新增页面/组件如果被路由或其他文件引用，要确保新文件也被加入构建上下文。
- `ModelPricingPanel.vue` 当前没有被引用，不提交不会影响构建。
- `ProfitView.vue` 已被路由引用，不提交或不带入构建上下文会导致前端构建失败。
