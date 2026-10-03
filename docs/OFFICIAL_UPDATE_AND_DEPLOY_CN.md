# ANLAPI 维护者升级与部署指南

> 本文是维护者使用的技术资料，不是项目首页介绍。它记录版本升级、差异核对、构建、部署和回滚方法；产品能力请以根目录 README 为准。

本文用于维护当前 ANLAPI：按功能批次同步兼容性、安全和运行时更新，同时保留 ANLAPI 的界面、支付、生图、账号隔离、用量计费和用户级并发等产品能力。

## 当前基线

当前产品版本为 `ANLAPI 1.0.11`，官方功能基线为 `Sub2API v0.2.13`。具体版本以根目录 README、`backend/cmd/server/VERSION` 和发布分支为准。

当前公开发布分支是：

```bash
release/anlapi-v0.2.13-public
```

公开发布分支用于交付已验证版本；下一次升级应从独立工作分支开始，完成差异核对、定向回移和验证后再发布。不要在交付分支上直接进行大范围同步。

后续更新应以相邻官方 tag 的发布差异为依据，按功能回移到 ANLAPI 工作分支，并保留支付、生图、账号隔离、OAuth/API Key 隔离和用户级并发策略。

## 远程与保护设置

当前私有主仓库使用 `origin`，Sub2API 官方仓库使用只读同步远程 `sub2api`。操作前先确认：

```bash
git remote -v
git remote set-url sub2api https://github.com/Wei-Shaw/sub2api.git
git fetch sub2api --tags --prune
```

建议开启 Git 冲突复用记录，减少后续重复处理同类冲突：

```bash
git config rerere.enabled true
```

## 每次同步官方新版本

当前维护主线使用以下流程：

1. 确认工作区状态，并为未提交改动保留独立快照。
2. 执行 `git fetch sub2api --tags --prune`，再通过 GitHub Releases API 确认最新正式版本。
3. 使用 `git diff <上一官方标签>..<新官方标签>` 和发布说明拆分功能、修复、安全变更及迁移。
4. 逐功能回移到 ANL 代码结构；禁止覆盖 ANL 的支付、账号权限、分组、生图和并发定制。
5. 对受影响包运行定向测试，再运行后端全量测试、前端类型检查、Lint、Vitest、构建、UTF-8 与乱码扫描。
6. 只有验证通过后才能提交、推送和进入生产部署流程。

下面的直接合并流程只适用于确实保留上游祖先关系的工作分支。实际操作时，将 `<target-tag>`、`<previous-tag>` 和 `<target-build>` 替换为经过核验的目标版本。

### 1. 确认工作区干净

```bash
git switch main
git status --short
```

必须看到没有未提交改动后再继续。若有本地改动，先提交或另建分支保存。

### 2. 拉取官方 tag

```bash
git fetch upstream --tags --prune
```

查看最近版本：

```bash
git tag -l "v0.1.*" --sort=-version:refname | head
```

Windows PowerShell 可用：

```powershell
git tag -l "v0.1.*" --sort=-version:refname | Select-Object -First 10
```

如果只想拉取指定 tag：

```bash
git fetch upstream refs/tags/<target-tag>:refs/tags/<target-tag>
```

### 3. 创建升级分支

```bash
git switch main
git pull --ff-only
git switch -c upgrade/<target-tag>-integration
```

如果没有远程私有仓库，`git pull --ff-only` 可以跳过。

### 4. 合并官方版本

```bash
git merge --no-ff <target-tag>
```

如果没有冲突，直接进入验证步骤。

如果出现冲突，先查看冲突文件：

```bash
git status --short
git diff --name-only --diff-filter=U
```

### 5. 冲突处理原则

处理冲突时优先保留二开业务边界：

- 普通用户账号必须继续按 `owner_user_id`、`share_mode`、`share_status` 做后端隔离。
- 公共账号分组只能作为调度池，不能成为权限边界。
- 普通用户不能通过 API Key、Upstream、URL、Base URL 添加账号。
- 凭证导入必须继续拒绝 API Key-like、URL/base_url/upstream/endpoint/host/proxy_url、cookie、authorization、AWS key 等敏感格式。
- 账号主自己调用自己的公共账号不能产生分成收入。
- 分成流水必须保持幂等、可审计。

以下文件属于高耦合区域，官方版本变更时需要重点复核：

```bash
backend/cmd/server/wire_gen.go
backend/internal/service/wire.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/setting_service.go
frontend/src/components/account/CreateAccountModal.vue
frontend/src/types/index.ts
frontend/src/views/admin/AccountsView.vue
```

如果官方再次修改这些文件，重点检查：

- `wire.go` 和 `wire_gen.go`：官方新增依赖时，不要覆盖二开的账号共享、分成策略、凭证导入相关依赖注入。
- `openai_gateway_service.go`：不要丢失 `AccountSharePolicyRepository`、公共账号结算、自用排除逻辑。
- `setting_service.go`：不要丢失用户私有分组、分成策略、收益管理相关设置默认值。
- `CreateAccountModal.vue`：官方新增账号类型时，普通用户入口仍必须受 `isUserScope` 限制。
- `types/index.ts`：官方新增枚举时，要和二开的 `share_mode/share_status` 类型一起保留。
- `AccountsView.vue`：官方账号管理增强不能覆盖二开的凭证导入、归属/共享列、刷新和筛选逻辑。

处理完冲突后检查是否还有冲突标记：

```bash
rg -n "^(<<<<<<< .+|=======$|>>>>>>> .+)" backend frontend
git diff --name-only --diff-filter=U
```

如果改到了 Wire 注入关系，重新生成 Wire：

```bash
cd backend
go generate ./cmd/server
cd ..
```

格式化 Go 文件：

```bash
gofmt -w backend/cmd/server/wire_gen.go backend/internal/service/*.go backend/internal/handler/**/*.go backend/internal/repository/*.go
```

如果 PowerShell 对 `**` 展开不符合预期，可以只格式化实际改动的 `.go` 文件。

### 6. 验证

后端：

```bash
cd backend
go test ./...
cd ..
```

前端：

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run typecheck
cd ..
```

建议再检查一次缓存区和冲突标记：

```bash
git diff --check
rg -n "^(<<<<<<< .+|=======$|>>>>>>> .+)" backend frontend
```

如果合并已暂存，使用：

```bash
git diff --cached --check
```

### 7. 提交合并

```bash
git status
git add <已解决的文件>
git commit -m "release: integrate <target-tag>"
```

合并完成后，把升级分支合回长期二开主线：

```bash
git switch main
git merge --ff-only upgrade/<target-tag>-integration
git push
```

如果 `custom/main` 没有远程仓库，最后的 `git push` 跳过。

## 合并后的重点人工检查

自动测试通过后，建议在测试环境做以下人工检查：

- 管理员账号管理：平台账号、用户私有、用户公共、校验状态、归属/共享列展示正常。
- 普通用户我的账号：只能看到自己的账号，不能编辑或删除他人账号。
- 普通用户新增账号：页面上只能走 OAuth 或凭证导入，后端也拒绝 API Key、Upstream、URL/Base URL。
- 凭证导入：用户接口和管理员接口都能识别合法凭证，能拒绝敏感字段。
- 公共账号调用：别人调用用户公共账号时生成分成，账号主自用时不生成分成。
- 收益管理：消费用户、账号主收益、分组/账号/模型的金额方向不混淆。

测试环境可以写入测试数据；生产数据库不要直接手工新增、修改、删除数据。

## 自定义镜像构建

生产环境不要继续使用官方镜像 `anlapi:latest`，否则会覆盖二开功能。必须构建并使用自己的镜像。

根目录 `Dockerfile` 是推荐的生产镜像构建入口，会完成：

- 前端 `pnpm run build`
- 后端 `go build -tags embed`
- 前端产物嵌入后端
- 最终生成运行镜像

### 本机单架构构建

Linux/macOS：

```bash
VERSION=<target-build>
COMMIT=$(git rev-parse --short HEAD)
IMAGE=registry.example.com/anlapi-custom:$VERSION

docker build \
  --build-arg VERSION=$VERSION \
  --build-arg COMMIT=$COMMIT \
  -t $IMAGE \
  -t registry.example.com/anlapi-custom:latest \
  .
```

Windows PowerShell：

```powershell
$version = "<target-build>"
$commit = git rev-parse --short HEAD
$image = "registry.example.com/anlapi-custom:$version"

docker build `
  --build-arg VERSION=$version `
  --build-arg COMMIT=$commit `
  -t $image `
  -t registry.example.com/anlapi-custom:latest `
  .
```

### 多架构构建并推送

如果服务器可能是 `amd64` 或 `arm64`，使用 `buildx`：

```bash
VERSION=<target-build>
COMMIT=$(git rev-parse --short HEAD)
IMAGE=registry.example.com/anlapi-custom:$VERSION

docker buildx create --use --name anlapi-builder || true
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$VERSION \
  --build-arg COMMIT=$COMMIT \
  -t $IMAGE \
  -t registry.example.com/anlapi-custom:latest \
  --push \
  .
```

如果只在当前机器构建并推送单架构镜像：

```bash
docker login registry.example.com
docker push registry.example.com/anlapi-custom:<target-build>
docker push registry.example.com/anlapi-custom:latest
```

镜像 tag 建议包含官方版本和二开构建序号，例如：

```bash
<target-build>
<next-build>
```

不要只依赖 `latest`，否则回滚时无法准确定位版本。

## 服务器部署方式

推荐生产环境使用：

```bash
deploy/docker-compose.local.yml
```

原因：

- `data`、`postgres_data`、`redis_data` 都在部署目录下，备份和迁移直观。
- 不依赖 Docker 命名卷路径。
- 整个 `deploy` 目录可以打包迁移。

如果你已经使用 `deploy/docker-compose.yml` 的命名卷方式，也可以继续使用，但备份和迁移要额外处理 Docker volumes。

## 首次部署自定义镜像

服务器目录示例：

```bash
/opt/anlapi
```

准备部署文件：

```bash
mkdir -p /opt/anlapi
cd /opt/anlapi
```

把仓库中的 `deploy/docker-compose.local.yml` 和 `deploy/.env.example` 上传到服务器。

```bash
cp .env.example .env
```

编辑 `.env`，至少修改：

```bash
POSTGRES_PASSWORD=<强密码>
JWT_SECRET=<固定随机密钥>
TOTP_ENCRYPTION_KEY=<固定随机密钥>
ADMIN_EMAIL=<管理员邮箱>
ADMIN_PASSWORD=<首次部署可设置，已有数据后不要随意改>
TZ=Asia/Shanghai
```

生成密钥示例：

```bash
openssl rand -hex 32
```

建议使用 `docker-compose.override.yml` 指定自定义镜像，避免直接改官方 compose 文件：

```yaml
services:
  anlapi:
    image: registry.example.com/anlapi-custom:<target-build>
```

启动：

```bash
docker login registry.example.com
docker compose -f docker-compose.local.yml -f docker-compose.override.yml pull
docker compose -f docker-compose.local.yml -f docker-compose.override.yml up -d
docker compose -f docker-compose.local.yml -f docker-compose.override.yml ps
docker compose -f docker-compose.local.yml -f docker-compose.override.yml logs -f anlapi
```

健康检查：

```bash
curl -fsS http://127.0.0.1:8080/health
```

如果 `.env` 中修改了 `SERVER_PORT`，把 `8080` 替换成实际端口。

## 已部署环境更新镜像

以下流程用于把服务器从旧自定义镜像更新到新自定义镜像。

### 1. 备份

进入服务器部署目录：

```bash
cd /opt/anlapi
mkdir -p backups
```

本地目录版备份：

```bash
tar czf backups/anlapi-files-$(date +%F-%H%M%S).tgz \
  .env docker-compose.local.yml docker-compose.override.yml data postgres_data redis_data
```

再做一次 PostgreSQL 逻辑备份：

```bash
set -a
. ./.env
set +a

docker compose -f docker-compose.local.yml -f docker-compose.override.yml exec -T postgres \
  pg_dump -U "${POSTGRES_USER:-anlapi}" "${POSTGRES_DB:-anlapi}" \
  > backups/anlapi-db-$(date +%F-%H%M%S).sql
```

不要执行：

```bash
docker compose down -v
```

`down -v` 会删除数据卷。生产环境除非已经确认要清空数据，否则禁止使用。

### 2. 修改镜像 tag

编辑服务器上的 `docker-compose.override.yml`：

```yaml
services:
  anlapi:
    image: registry.example.com/anlapi-custom:<target-build>
```

### 3. 拉取并重建容器

```bash
docker login registry.example.com
docker compose -f docker-compose.local.yml -f docker-compose.override.yml pull anlapi
docker compose -f docker-compose.local.yml -f docker-compose.override.yml up -d anlapi
```

只更新应用容器时，不需要重建 PostgreSQL 和 Redis。

### 4. 验证

```bash
docker compose -f docker-compose.local.yml -f docker-compose.override.yml ps
docker compose -f docker-compose.local.yml -f docker-compose.override.yml logs --tail=200 anlapi
curl -fsS http://127.0.0.1:8080/health
```

后台页面验证：

- 管理员登录正常。
- API Key 调用正常。
- 账号列表、我的账号、收益管理页面能正常加载。
- 新版本迁移日志没有报错。

## 回滚

回滚前先确认旧镜像 tag，例如：

```bash
registry.example.com/anlapi-custom:<previous-build>
```

修改 `docker-compose.override.yml`：

```yaml
services:
  anlapi:
    image: registry.example.com/anlapi-custom:<previous-build>
```

执行：

```bash
docker compose -f docker-compose.local.yml -f docker-compose.override.yml pull anlapi
docker compose -f docker-compose.local.yml -f docker-compose.override.yml up -d anlapi
docker compose -f docker-compose.local.yml -f docker-compose.override.yml logs --tail=200 anlapi
```

注意：如果新版本已经执行了不可逆数据库迁移，单纯回滚镜像可能不够。此时要结合升级前的数据库备份恢复。恢复生产数据库属于高风险操作，必须先确认影响范围和恢复点。

## 服务器直接源码构建

不推荐生产环境在服务器直接从源码构建。推荐流程是本地或 CI 构建镜像、推送镜像仓库、服务器只拉取镜像。

如果临时需要在服务器源码构建测试，可以使用开发 compose：

```bash
cd deploy
docker compose -f docker-compose.dev.yml up -d --build
docker compose -f docker-compose.dev.yml logs -f anlapi
```

这个方式适合测试，不建议作为正式生产部署方式。

## 常见问题

### 误用了官方镜像怎么办

如果 `docker-compose.local.yml` 或 `docker-compose.override.yml` 中仍是：

```yaml
image: anlapi:latest
```

说明正在使用官方镜像，不包含二开功能。改成自己的镜像：

```yaml
    image: registry.example.com/anlapi-custom:<target-build>
```

然后重新拉取并启动：

```bash
docker compose -f docker-compose.local.yml -f docker-compose.override.yml pull anlapi
docker compose -f docker-compose.local.yml -f docker-compose.override.yml up -d anlapi
```

### 合并后如何确认二开差异还在

对比当前分支和官方 tag：

```bash
git diff --stat <previous-tag>..<target-tag>
git diff --name-status <previous-tag>..<target-tag>
```

重点确认这些二开模块仍存在：

```bash
backend/internal/handler/user_account_handler.go
backend/internal/service/account_credential_import.go
backend/internal/handler/admin/account_share_policy_handler.go
backend/internal/repository/account_share_policy_repo.go
frontend/src/views/user/AccountsView.vue
frontend/src/components/account/CredentialImportModal.vue
frontend/src/components/admin/revenue/SharePolicyPanel.vue
frontend/src/components/admin/revenue/ShareSettlementsPanel.vue
```

### 合并时发现官方大改了账号或计费模块

不要为了通过编译删除二开逻辑。先停止合并并记录：

```bash
git status --short
git diff --name-only --diff-filter=U
```

然后重点审查：

- 账号权限边界是否仍由后端字段控制。
- 调度池是否会暴露私有账号或未通过公共账号。
- `usage_log_id/request_id` 是否仍能保证分成幂等。
- 消费用户扣费、账号主入账、平台净收益的金额方向是否被官方改动影响。
- 凭证导入的禁止字段校验是否被覆盖。

确认清楚后再继续解决冲突。

### 更新后登录全部失效

检查 `.env` 中是否固定设置：

```bash
JWT_SECRET
TOTP_ENCRYPTION_KEY
```

这两个值不能每次启动随机变化。已有生产环境不要随意改。

## 发布前检查清单

合并阶段：

- `git status --short` 干净。
- `go test ./...` 通过。
- `pnpm run typecheck` 通过。
- 没有 Git 冲突标记。
- 二开功能重点文件仍存在。

镜像阶段：

- 镜像 tag 包含官方版本和二开构建号。
- 镜像已推送到自己的镜像仓库。
- 服务器 compose 使用自定义镜像，不使用 `anlapi:latest`。

部署阶段：

- 升级前已备份部署目录和 PostgreSQL。
- 没有执行 `docker compose down -v`。
- `.env` 中生产密钥固定。
- 更新后 `/health` 正常。
- 管理后台、账号调用、收益管理核心页面验证正常。
