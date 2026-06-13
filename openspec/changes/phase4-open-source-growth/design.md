## Context

Golem 当前已经具备较完整的 agent 基础能力与多 provider 支持，但项目仍处于“工程能力大于产品吸引力”的阶段。现有 active changes 主要聚焦质量、运行时能力和生产就绪性，而新的增长目标要求项目在以下维度同时变强：

- **可运行**：新用户必须能在几分钟内完成安装、配置和首个成功案例
- **可理解**：学习路径要从 README、Quick Start、Tutorial、Architecture 到 Troubleshooting 自然递进
- **可传播**：官网、截图、演示、benchmark、showcase、release notes 需要形成对外转化资产
- **可沉迷**：首次使用就要看到“它真能帮我完成事情”的反馈回路，而不只是展示底层架构

约束不变：继续保持纯 Go、单静态二进制、CGO_ENABLED=0、Linux/Termux 友好、教育项目定位与可维护分层架构。

## Goals / Non-Goals

**Goals:**
- 定义一个按 phase 推进的产品化/开源增长方案，覆盖运行路径、文档体系、官网体验、用户激活和传播资产
- 将“文档、官网、产品首用体验、GitHub 增长”纳入一个统一执行框架，而不是分散优化
- 优先解决“项目能跑起来、能被快速理解、能被快速分享”这三个增长前提
- 为后续实现提供明确的 phase 划分与任务边界，避免与已有 phase1/2/3 变更冲突

**Non-Goals:**
- 本 change 不直接实现所有业务代码，只定义下一阶段的需求与实施蓝图
- 不把 star 增长等同于营销动作堆砌；重点仍是产品可用性和用户价值
- 不改变当前纯 Go / 单二进制 / 多 provider / 学习型项目的根定位
- 不在本阶段引入重型前端框架或复杂 SaaS 化后端作为前置条件

## Decisions

### D1. 下一阶段以“产品化增长”而不是“继续补底层能力”为主线
- **选择**：将下一阶段定义为 open-source growth phase，而不是单一技术能力 phase
- **原因**：当前项目的核心差距不再是完全缺能力，而是缺“能被陌生开发者快速感知价值”的包装与体验
- **替代方案**：继续先做 metrics/security/failover 等底层工作
- **放弃原因**：这些工作重要，但更适合作为 phase3 的生产强化，不足以单独推动 adoption

### D2. 用四条并行产品轨推进：Run、Learn、Love、Share
- **选择**：将方案拆成四条主轨：
  1. **Run**：安装-配置-运行-验证成功
  2. **Learn**：教程与架构文档
  3. **Love**：首用体验、demo、模板、wow moment
  4. **Share**：官网首页、README、benchmark、showcase、release assets
- **原因**：这四条轨道分别对应新用户转化漏斗中的激活、理解、留存、传播
- **替代方案**：按代码目录或模块拆分 phase
- **放弃原因**：模块化拆分更适合内部交付，不适合围绕用户转化组织工作

### D3. 官方站点采用“首页转化 + 教程导航 + 架构深潜 + 证据资产”结构
- **选择**：站点结构参考你给出的 claude-code-Go 页面风格，但内容组织围绕 Golem 的差异化卖点：纯 Go、单二进制、Termux 友好、多通道、多 provider、教育价值
- **信息架构**：
  - Home：价值主张、视觉 demo、核心能力、快速开始
  - Guide：安装/配置/首用成功/教程
  - Architecture：分层架构、agent loop、provider/tool/session 机制
  - Proof：benchmark、showcase、roadmap、troubleshooting
- **替代方案**：先只写 README，不单独规划官网
- **放弃原因**：README 无法承载完整教程和转化内容，官网是 1k+ star 目标的重要放大器

### D4. 文档优先级按“激活路径”排序，而不是按模块排序
- **选择**：文档首先服务新用户在 5 分钟内成功运行，然后再逐步进入进阶教程和架构说明
- **推荐顺序**：
  1. Quick Start
  2. Install Matrix (Linux/macOS/Windows/Termux)
  3. First Success Path
  4. Common Pitfalls / Troubleshooting
  5. Tutorial Series
  6. Architecture Deep Dive
  7. Contribution Guide
- **替代方案**：先补全架构文档
- **放弃原因**：架构深潜对增长有帮助，但不能替代首用成功率

### D5. “让用户上瘾”优先通过首用闭环实现，而不是复杂功能堆叠
- **选择**：将上瘾感来源定义为“迅速得到正确帮助 + 可复现 demo + 持续探索入口”
- **机制**：
  - first-run example commands
  - 内置可复制 demo prompt
  - 示例配置与 showcase
  - session continuity / 历史恢复 / 模板任务
- **替代方案**：增加更多高级功能作为 wow factor
- **放弃原因**：没有稳定激活闭环时，高级功能只会提高理解成本

### D6. 路线图按 phase 组织，但优先完成单点“可展示成果”
- **选择**：每个 phase 都要产出一个用户可见、可传播的成果，而不是只完成内部重构
- **建议 phase**：
  - **Phase A**: Runnable foundation
  - **Phase B**: Premium docs and learning journey
  - **Phase C**: Official site and conversion assets
  - **Phase D**: Activation, retention, community flywheel
- **原因**：GitHub star 增长依赖持续、可见的用户价值释放

## Risks / Trade-offs

- **范围过大** → 通过 phase 划分和按用户漏斗组织任务，避免一次性铺太大
- **官网和文档投入高，但代码收益不直接** → 把它视作开源产品的一部分，而不是附属品
- **增长导向可能稀释“学习型项目”定位** → 所有教程与官网内容保留“为什么这样设计”的教学维度
- **已有 phase3-production-readiness 尚未启动，可能与新 phase 并行冲突** → 新 change 只定义下一步增长方案，不替代 phase3 的生产强化任务
- **“上瘾”目标容易流于主观** → 用激活时间、首次成功率、文档完成率、站点转化点击等可观察指标约束

## Migration Plan

1. 先完成当前 `project-review-and-roadmap` 中剩余的硬功能任务，确保基础能力补齐
2. 基于本 change 启动下一阶段实施时，优先落地 Phase A（项目可运行 + 首次成功路径）
3. 然后执行 Phase B / C：文档体系与官网体验同步推进
4. 最后执行 Phase D：showcase、社区、发布节奏、用户留存机制
5. 如果资源不足，可按“Run → Learn → Share → Love”优先级裁剪

## Open Questions

- 官网内容源最终放在当前仓库、独立 docs 仓库，还是 monorepo 子目录？
- 是否需要将中英文文档在本阶段同时纳入范围，还是先英文主站后补中文？
- 安装分发主入口优先 `go install`、release binary，还是 shell install script？
- 是否要在本阶段引入 demo recording / benchmark 自动化流水线？
