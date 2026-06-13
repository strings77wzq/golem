## Why

Golem 已经具备较完整的 AI agent 基础能力，但距离“开发者愿意安装、能快速跑起来、愿意持续使用并主动传播”的 1k+ star 开源项目还有明显差距。下一阶段的核心不再只是补技术能力，而是把项目打磨成一个**可立即运行、文档体验优秀、官网具有说服力、首次使用有惊喜感**的产品化开源项目。

## What Changes

- 建立一个以“先跑起来”为首要目标的 runnable path：安装、配置、启动、首个成功案例必须在几分钟内完成
- 重构开发者学习路径：从 README、Quick Start、Tutorial、Architecture 到 Troubleshooting 形成渐进式上手闭环
- 设计并落地官方站点的信息架构、首页叙事、能力展示、教程导航与中英文内容策略
- 增强首次使用与持续使用体验：提供 demo flows、模板命令、示例配置、内置 showcase，让用户更容易形成“上瘾式”反馈循环
- 建立面向 GitHub 增长的开源包装层：版本发布节奏、截图/演示资产、对比页、benchmark、showcase、贡献入口
- 对现有 agent 能力做必要的产品化补充，使其更适合作为面向开发者的公开项目被理解、试用和传播

## Capabilities

### New Capabilities
- `runnable-onboarding`: 规范安装、配置、运行、验证成功的首条路径，确保新用户能快速跑通项目
- `developer-learning-journey`: 建立高质量的分层教学文档体系，覆盖快速开始、教程、架构、排障、贡献指南
- `official-site-experience`: 定义官方站点首页、导航、内容层级、双语策略、演示区块和转化路径
- `activation-and-retention`: 设计首用体验、内置示例、demo 场景、习惯形成机制与用户反馈回路
- `open-source-growth-assets`: 建立 README、发布素材、截图、benchmark、showcase、对比内容和社区转化资产

### Modified Capabilities
- `agent`: 调整对外呈现方式和首用行为，使 agent 更适合作为公开产品被试用、理解和推广

## Impact

- 影响代码：`cmd/golem/`, `internal/channels/`, `internal/gateway/`, `core/agent/`, `feature/skills/`
- 影响文档：`README.md`, `docs/`, `docs/study/`, `openspec/changes/`, 以及未来的站点内容源文件
- 影响官网：需要新增或重构站点首页、教程索引、架构说明、benchmark、showcase、troubleshooting 等页面
- 影响发布：需要明确安装方式、二进制分发、演示素材、版本说明和贡献入口
- 依赖约束不变：继续保持 `CGO_ENABLED=0`、纯 Go、单静态二进制方向
