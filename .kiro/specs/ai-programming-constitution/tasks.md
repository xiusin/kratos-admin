# Implementation Plan: AI Programming Constitution

## Overview

本实施计划将 AI 编程规范（AI Programming Constitution）功能分解为可执行的开发任务。该功能为 GO + Vue 后台管理框架项目建立完整的 AI 辅助编程约束系统，包括规范文档、任务留痕、代码验证、防幻觉机制、回滚系统和文档同步功能。

技术栈：
- 后端：Go 语言
- 前端：Vue 3 Composition API
- 验证工具：gofmt, golangci-lint, eslint, vue-tsc, protoc
- 测试框架：Go testing + gopter (属性测试)

## Tasks

- [x] 1. 创建项目基础结构和配置文件
  - 创建 `.ai/` 目录结构（constitution.md, config.yaml, traces/, templates/）
  - 创建 Go 模块和包结构（pkg/constitution/）
  - 定义核心数据类型和接口（TaskTrace, ValidationResult, Decision 等）
  - _Requirements: 6.5, 6.6_

- [ ] 2. 实现 Constitution 文档和配置系统
  - [x] 2.1 创建 Constitution 主文档模板
    - 编写 `.ai/constitution.md` 包含所有必需章节（角色定义、代码规范、架构约束、防幻觉规则、禁止行为、任务协议、验证要求、回滚机制、文档要求、安全性能）
    - 定义每个章节的详细内容和规则
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 4.1, 4.2, 4.3, 4.4, 4.5, 4.7, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8, 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 12.7, 12.8_

  - [x] 2.2 创建配置文件
    - 编写 `.ai/config.yaml` 定义工具路径、验证规则、留痕配置、回滚配置
    - 配置 Go 验证工具（gofmt, golangci-lint）
    - 配置 Vue 验证工具（eslint, vue-tsc, prettier）
    - 配置 Protobuf 和 Ent 工具
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [x] 2.3 实现配置加载器
    - 实现 Go 代码读取和解析 config.yaml
    - 实现配置验证逻辑
    - 实现配置热加载功能
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [x] 3. 实现任务留痕系统（Task Trace System）
  - [x] 3.1 实现 TaskTraceManager 接口
    - 实现 CreateTask 方法（创建任务记录，生成 UUID，初始化 JSON 文件）
    - 实现 RecordDecision 方法（记录决策和理由）
    - 实现 RecordCodeChange 方法（记录文件变更）
    - 实现 RecordValidation 方法（记录验证结果）
    - 实现 AddReference 方法（添加参考资料）
    - 实现 CompleteTask 和 FailTask 方法
    - 实现 RecordRollback 方法
    - 实现 GetTask 和 ListTasks 方法
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.6, 6.7_

  - [ ]* 3.2 编写 TaskTraceManager 单元测试
    - 测试任务创建和 JSON 格式
    - 测试决策记录和代码变更记录
    - 测试文件存储位置和命名
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [ ]* 3.3 编写任务留痕完整性属性测试
    - **Property 1: Task Trace Completeness**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.7**
    - 使用 gopter 验证所有任务记录包含必需字段
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.7_

- [x] 4. 实现代码验证器（Code Validator）
  - [x] 4.1 实现 CodeValidator 接口
    - 实现 ValidateGoCode 方法（调用 gofmt 和 golangci-lint）
    - 实现 ValidateVueCode 方法（调用 eslint 和 vue-tsc）
    - 实现 ValidateProtobuf 方法（调用 protoc）
    - 实现 ValidateEntSchema 方法（调用 ent generate）
    - 实现 ValidateImports 方法（检查导入包是否存在）
    - 实现 RunTests 方法（运行相关测试）
    - 实现工具输出解析和错误提取
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.7_

  - [ ]* 4.2 编写 CodeValidator 单元测试
    - 测试 Go 代码验证（有效代码和语法错误）
    - 测试 Vue 代码验证
    - 测试 Protobuf 和 Ent Schema 验证
    - 测试导入检查
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.7_

  - [ ]* 4.3 编写代码验证执行属性测试
    - **Property 2: Code Validation Execution**
    - **Validates: Requirements 7.1, 7.2, 7.3, 7.4**
    - 验证所有代码生成操作都触发相应的验证器
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [ ]* 4.4 编写验证失败阻止完成属性测试
    - **Property 3: Validation Failure Prevents Completion**
    - **Validates: Requirements 7.6**
    - 验证包含验证错误的任务无法标记为完成
    - _Requirements: 7.6_

- [x] 5. 实现防幻觉验证器（Anti-Hallucination Verifier）
  - [x] 5.1 实现 AntiHallucinationVerifier 接口
    - 实现 VerifyAPIExists 方法（解析 .proto 文件，构建 API 索引）
    - 实现 VerifyFunctionExists 方法（使用 AST 解析器扫描代码库）
    - 实现 VerifyModuleExists 方法（检查 go.mod 和 package.json）
    - 实现 VerifyConfigKeyExists 方法（检查配置文件）
    - 实现 GetAPIReference 和 GetFunctionSignature 方法
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.7_

  - [x] 5.2 实现索引数据库
    - 实现 Protobuf API 索引构建
    - 实现函数和类型索引构建
    - 实现配置键索引
    - 实现依赖包索引
    - 实现索引更新触发机制
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [ ]* 5.3 编写防幻觉验证单元测试
    - 测试 API 存在性验证（存在和不存在的情况）
    - 测试函数存在性验证
    - 测试模块存在性验证
    - 测试配置键验证
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [ ]* 5.4 编写未验证元素确认属性测试
    - **Property 7: Unverified Element Confirmation**
    - **Validates: Requirements 4.6**
    - 验证无法验证的代码元素会请求开发者确认
    - _Requirements: 4.6_

- [x] 6. 实现回滚机制（Rollback Mechanism）
  - [x] 6.1 实现 RollbackManager 接口
    - 实现 CreateBackup 方法（创建文件备份，计算哈希）
    - 实现 Rollback 方法（恢复文件，处理冲突）
    - 实现 GetBackup 和 ListBackups 方法
    - 实现 CleanupOldBackups 方法（清理过期备份）
    - _Requirements: 8.3, 8.4, 8.7_

  - [x] 6.2 实现回滚触发逻辑
    - 实现验证失败触发回滚
    - 实现规范违反检测和触发回滚
    - 实现手动回滚触发
    - 集成 TaskTraceManager 记录回滚信息
    - _Requirements: 8.1, 8.2, 8.4, 8.5, 8.6_

  - [ ]* 6.3 编写回滚机制单元测试
    - 测试文件备份和恢复
    - 测试回滚原因记录
    - 测试冲突处理
    - _Requirements: 8.3, 8.4, 8.5_

  - [ ]* 6.4 编写回滚触发属性测试
    - **Property 4: Rollback Trigger on Violation**
    - **Validates: Requirements 8.1, 8.2**
    - 验证规范违反和严重错误自动触发回滚
    - _Requirements: 8.1, 8.2_

  - [ ]* 6.5 编写回滚完整性属性测试
    - **Property 5: Rollback Completeness**
    - **Validates: Requirements 8.3, 8.4, 8.5**
    - 验证回滚恢复所有文件并记录原因
    - _Requirements: 8.3, 8.4, 8.5_

- [ ] 7. Checkpoint - 核心后端功能验证
  - 确保所有测试通过，核心功能正常工作，如有问题请向用户询问

- [x] 8. 实现文档同步器（Documentation Syncer）
  - [x] 8.1 实现 DocumentationSyncer 接口
    - 实现 SyncAPIDocumentation 方法（从 .proto 文件提取注释和定义）
    - 实现 SyncComponentDocumentation 方法（从 Vue 组件提取 props、events、slots）
    - 实现 SyncFeatureDocumentation 方法（生成功能文档）
    - 实现 GenerateAPIReference 方法（生成完整 API 参考）
    - 实现 ValidateDocumentation 方法（检查文档完整性）
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7_

  - [x] 8.2 创建文档模板
    - 创建 API 文档模板（`.ai/templates/api-doc.tmpl`）
    - 创建组件文档模板（`.ai/templates/component-doc.tmpl`）
    - 创建功能文档模板（`.ai/templates/feature-doc.tmpl`）
    - _Requirements: 11.5, 11.6_

  - [ ]* 8.3 编写文档同步单元测试
    - 测试 API 文档生成
    - 测试组件文档生成
    - 测试文档完整性验证
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

  - [ ]* 8.4 编写文档同步属性测试
    - **Property 6: Documentation Synchronization**
    - **Validates: Requirements 11.1, 11.2, 11.3, 11.4**
    - 验证代码变更自动更新对应文档
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

- [x] 9. 实现规范违反检测器（Constitution Violation Detector）
  - [x] 9.1 实现禁止行为检测
    - 实现架构修改检测（跨层调用、目录结构变更）
    - 实现安全违规检测（绕过认证、硬编码密钥、日志泄露）
    - 实现依赖管理违规检测（未批准的依赖）
    - 实现 Schema 修改违规检测（删除迁移、破坏兼容性）
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8_

  - [x] 9.2 实现规范规则引擎
    - 解析 Constitution 文档中的规则
    - 实现规则匹配和违规判定
    - 实现严重程度评估
    - 生成违规报告和修复建议
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8_

  - [ ]* 9.3 编写规范违反检测单元测试
    - 测试各类禁止行为检测
    - 测试规则引擎匹配
    - 测试严重程度评估
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8_

- [-] 10. 实现错误处理和报告系统
  - [x] 10.1 实现错误分类和处理
    - 实现验证错误处理（语法、lint、类型、编译、测试）
    - 实现规范违反错误处理
    - 实现幻觉错误处理
    - 实现系统错误处理（文件系统、工具执行、配置、网络）
    - _Requirements: 7.6, 8.1, 8.2, 8.5_

  - [x] 10.2 实现错误恢复策略
    - 实现自动重试机制（配置重试次数、退避策略）
    - 实现优雅降级
    - 实现人工介入流程
    - _Requirements: 8.5_

  - [x] 10.3 实现错误报告生成
    - 实现结构化错误报告（JSON 格式）
    - 包含错误详情、代码片段、修复建议
    - 引用 Constitution 相关章节
    - _Requirements: 8.5_

  - [ ]* 10.4 编写错误处理单元测试
    - 测试各类错误分类
    - 测试重试机制
    - 测试错误报告生成
    - _Requirements: 8.1, 8.2, 8.5_

- [~] 11. Checkpoint - 后端系统集成测试
  - 确保所有后端组件正确集成，端到端流程正常工作，如有问题请向用户询问

- [~] 12. 实现 Vue 前端管理界面
  - [ ] 12.1 创建任务留痕查看器组件
    - 创建 TaskTraceList.vue（任务列表，支持筛选和搜索）
    - 创建 TaskTraceDetail.vue（任务详情，显示决策、代码变更、验证结果）
    - 创建 TaskTraceTimeline.vue（任务时间线可视化）
    - 使用 Vue 3 Composition API 和 TypeScript
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [ ] 12.2 创建验证结果展示组件
    - 创建 ValidationResults.vue（验证结果列表，按严重程度分类）
    - 创建 ValidationError.vue（单个错误详情，代码高亮）
    - 支持错误过滤和排序
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.6_

  - [ ] 12.3 创建回滚管理组件
    - 创建 RollbackHistory.vue（回滚历史列表）
    - 创建 RollbackDetail.vue（回滚详情，显示恢复的文件）
    - 创建手动触发回滚的界面
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [ ] 12.4 创建 Constitution 文档查看器
    - 创建 ConstitutionViewer.vue（Markdown 渲染，章节导航）
    - 创建 ConstitutionSearch.vue（规则搜索功能）
    - 支持规则引用跳转
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [ ] 12.5 创建配置管理界面
    - 创建 ConfigEditor.vue（YAML 配置编辑器，语法高亮）
    - 创建配置验证和保存功能
    - 支持配置热加载
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [ ]* 12.6 编写 Vue 组件单元测试
    - 测试各组件的渲染和交互
    - 测试数据绑定和事件处理
    - 使用 Vitest 和 Vue Test Utils
    - _Requirements: 10.6_

- [~] 13. 实现 API 层和前后端集成
  - [ ] 13.1 定义 gRPC API
    - 创建 constitution.proto（定义 TaskTrace、Validation、Rollback 等服务）
    - 定义 GetTaskTrace、ListTaskTraces、TriggerRollback 等 RPC 方法
    - 生成 Go 和 TypeScript 代码
    - _Requirements: 3.2, 9.4_

  - [ ] 13.2 实现 gRPC 服务端
    - 实现 ConstitutionService（调用后端各组件）
    - 实现请求验证和错误处理
    - 实现认证和授权检查
    - _Requirements: 12.3, 12.4_

  - [ ] 13.3 实现前端 API 客户端
    - 创建 TypeScript gRPC 客户端封装
    - 实现请求拦截器和错误处理
    - 实现状态管理（Pinia store）
    - _Requirements: 10.5_

  - [ ]* 13.4 编写 API 集成测试
    - 测试完整的请求-响应流程
    - 测试错误处理和边界情况
    - _Requirements: 9.4, 12.3, 12.4_

- [~] 14. 创建代码模板和示例
  - [ ] 14.1 创建 Go 代码模板
    - 创建 `.ai/templates/go-service.tmpl`（gRPC 服务模板）
    - 创建 `.ai/templates/go-handler.tmpl`（业务逻辑处理器模板）
    - 创建 `.ai/templates/go-test.tmpl`（测试文件模板）
    - _Requirements: 2.3, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8_

  - [ ] 14.2 创建 Vue 代码模板
    - 创建 `.ai/templates/vue-component.tmpl`（Vue 组件模板）
    - 创建 `.ai/templates/vue-composable.tmpl`（Composable 函数模板）
    - 创建 `.ai/templates/vue-store.tmpl`（Pinia store 模板）
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8_

  - [ ] 14.3 创建示例代码
    - 创建符合规范的 Go 服务示例
    - 创建符合规范的 Vue 组件示例
    - 创建 Protobuf 和 Ent Schema 示例
    - _Requirements: 11.6_

- [~] 15. 实现 CLI 工具
  - [ ] 15.1 创建 constitution CLI 命令
    - 实现 `constitution init`（初始化 Constitution 文档和配置）
    - 实现 `constitution validate <file>`（验证单个文件）
    - 实现 `constitution check`（检查整个项目）
    - 实现 `constitution trace list`（列出任务留痕）
    - 实现 `constitution trace show <task-id>`（显示任务详情）
    - 实现 `constitution rollback <task-id>`（手动触发回滚）
    - 实现 `constitution index rebuild`（重建索引数据库）
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 6.1, 6.2, 6.3, 6.4, 7.1, 7.2, 7.3, 7.4, 8.1, 8.2_

  - [ ]* 15.2 编写 CLI 集成测试
    - 测试各命令的执行和输出
    - 测试错误处理
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [ ] 16. 编写完整的端到端测试
  - [ ]* 16.1 编写完整任务工作流测试
    - 测试从任务创建到完成的完整流程
    - 测试代码生成、验证、文档同步
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 7.1, 7.2, 11.1, 11.2_

  - [ ]* 16.2 编写回滚工作流测试
    - 测试违规检测和自动回滚流程
    - 测试文件恢复和状态一致性
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [ ]* 16.3 编写防幻觉工作流测试
    - 测试引用不存在元素的检测和确认流程
    - 测试索引构建和查询
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_

- [ ] 17. 创建项目文档
  - [ ] 17.1 编写用户指南
    - 编写 Constitution 系统使用指南
    - 编写 CLI 工具使用文档
    - 编写前端界面使用说明
    - _Requirements: 11.5_

  - [ ] 17.2 编写开发者文档
    - 编写架构设计文档
    - 编写 API 参考文档
    - 编写扩展和定制指南
    - _Requirements: 11.5, 11.6_

  - [ ] 17.3 编写部署和配置文档
    - 编写安装和配置指南
    - 编写工具链集成说明
    - 编写故障排查指南
    - _Requirements: 11.7_

- [ ] 18. Final Checkpoint - 完整系统验证
  - 运行所有测试（单元测试、属性测试、集成测试、端到端测试）
  - 验证所有功能正常工作
  - 检查代码质量和测试覆盖率
  - 如有问题请向用户询问

## Notes

- 标记 `*` 的任务为可选任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号，确保可追溯性
- Checkpoint 任务确保增量验证，及早发现问题
- 属性测试验证通用正确性属性，单元测试验证具体示例和边界情况
- 前端和后端可以并行开发，通过 Protobuf 定义的 API 进行集成
