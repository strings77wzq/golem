## 1. Phase A - Runnable foundation

- [x] 1.1 审计当前安装、配置、首次运行链路，列出 Linux amd64 与 Termux 的阻塞点
- [x] 1.2 统一并简化 Quick Start，使用户可在最短路径内完成 provider 配置与首次成功响应
- [x] 1.3 增加首用验证步骤（命令、预期输出、失败时回退路径）
- [x] 1.4 为主要平台补充安装矩阵与已知限制说明
- [x] 1.5 准备可复现的 first success demo 流程与示例命令

## 2. Phase B - Documentation excellence

- [x] 2.1 重写 README 顶部信息架构（价值主张、安装、快速开始、核心能力、文档入口）
- [x] 2.2 建立文档主导航：Introduction / Quick Start / Tutorials / Architecture / Troubleshooting / Contributing
- [x] 2.3 编写面向新用户的 5 分钟快速上手文档
- [x] 2.4 设计并编写渐进式教程序列（从基础使用到 tool calling / session / gateway）
- [x] 2.5 补充 Troubleshooting 文档，覆盖安装、配置、provider、Termux 和常见运行错误
- [x] 2.6 补充贡献者文档（CONTRIBUTING、issue 模板、开发环境说明、代码结构索引）

## 3. Phase C - Official site experience

- [x] 3.1 规划官网首页的信息架构，明确 hero、核心卖点、快速开始、证明材料与 CTA
- [x] 3.2 参考 claude-code-Go 风格，定义 Golem 官网视觉与内容模块映射
- [x] 3.3 规划站点双语结构（English / 中文）及主导航一致性策略
- [x] 3.4 增加教程入口页、架构入口页、benchmark/showcase/roadmap 页面结构
- [x] 3.5 为首页准备可展示的截图、终端录屏或 demo 资产清单

## 4. Phase D - Activation and retention

- [x] 4.1 设计首次启动后的激活路径：用户看到什么、先试什么、怎样得到 wow moment
- [x] 4.2 为首用体验准备内置示例 prompt、示例配置或示例工作流
- [x] 4.3 设计“下一步推荐”机制，把首次成功用户引导到更多教程或工作流
- [x] 4.4 明确哪些产品体验最能带来复用习惯（session continuity、模板任务、示例场景）

## 5. Phase E - Open-source growth assets

- [x] 5.1 定义 GitHub star 转化所需的对外资产：README、logo、badges、demo GIF、release notes、showcase
- [x] 5.2 规划 benchmark、comparison、showcase 三类证明材料的内容来源和展示形式
- [x] 5.3 规划版本发布节奏与 release artifact 策略（源码、二进制、平台覆盖）
- [x] 5.4 规划 GitHub Discussions / 社区入口 / 反馈收集方式

## 6. Phase F - Agent productization alignment

- [x] 6.1 审核当前 agent 对外体验，确定哪些行为需要为首用成功路径做产品化补充
- [x] 6.2 设计 agent 首次成功案例与官网/文档演示之间的统一叙事
- [x] 6.3 定义 agent 与教程、示例、showcase 之间的内容复用边界

## 7. Phase G - Rollout and success criteria

- [x] 7.1 定义本阶段衡量指标：安装成功率、首次成功时间、文档完成率、官网转化点击、GitHub star 增长
- [x] 7.2 将 Phase A-D 排成可执行里程碑，并给出建议先后顺序
- [x] 7.3 校验新路线图与现有 phase3-production-readiness、project-review-and-roadmap 的边界，避免重复实施
- [x] 7.4 完成最终路线图说明，作为后续 `/opsx-apply` 的执行入口
