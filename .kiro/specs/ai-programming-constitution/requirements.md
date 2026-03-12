# Requirements Document

## Introduction

本文档定义了 GO + Vue 后台管理框架项目的 AI 编程规范（项目宪法）。该规范作为约束 AI 辅助编程行为的核心文档，确保 AI 生成的代码符合项目架构、编码标准和设计原则，防止 AI 模型产生幻觉（hallucination），并建立完整的任务执行和审计机制。

## Glossary

- **AI_Agent**: 参与项目开发的 AI 助手或编程工具
- **Constitution_Document**: AI 编程规范文档，定义 AI 行为约束的核心文档
- **Hallucination**: AI 模型臆造不存在的 API、函数、模块或配置的现象
- **Task_Trace**: 任务留痕记录，包含 AI 执行的操作、决策依据和代码变更
- **Backend_System**: 基于 Go 语言的后端系统，使用 gRPC/Protobuf、Ent ORM 等技术
- **Frontend_System**: 基于 Vue 框架的前端系统
- **Code_Validator**: 代码验证器，用于检查生成代码的合规性
- **Architecture_Layer**: 架构层次，包括 api/、app/、pkg/ 等目录结构
- **Protobuf_Schema**: 使用 Protocol Buffers 定义的 API 接口规范
- **Ent_Schema**: 使用 Ent ORM 定义的数据模型规范
- **Event_Bus**: 事件总线系统，用于模块间解耦通信
- **Lua_Engine**: Lua 脚本引擎，提供系统扩展能力
- **Forbidden_Action**: 明确禁止 AI 执行的操作清单
- **Rollback_Mechanism**: 代码回滚机制，用于撤销不符合规范的变更

## Requirements

### Requirement 1: AI 角色定位与职责边界

**User Story:** 作为项目开发者，我希望明确定义 AI 的角色和职责边界，以便 AI 在正确的范围内提供帮助。

#### Acceptance Criteria

1. THE Constitution_Document SHALL define the AI_Agent role as a code assistant that generates code based on existing patterns
2. THE Constitution_Document SHALL specify that AI_Agent must not make architectural decisions without explicit approval
3. THE Constitution_Document SHALL list the responsibilities of AI_Agent including code generation, refactoring, and testing
4. THE Constitution_Document SHALL define the boundaries where AI_Agent must seek human approval before proceeding
5. THE Constitution_Document SHALL specify that AI_Agent must operate within the established project architecture

### Requirement 2: 代码生成规范

**User Story:** 作为项目开发者，我希望 AI 生成的代码遵循统一的命名、结构和注释规范，以便保持代码库的一致性。

#### Acceptance Criteria

1. THE Constitution_Document SHALL define naming conventions for Go backend code including package names, function names, and variable names
2. THE Constitution_Document SHALL define naming conventions for Vue frontend code including component names, method names, and prop names
3. THE Constitution_Document SHALL specify code structure requirements for Backend_System including Architecture_Layer organization
4. THE Constitution_Document SHALL specify comment requirements including function documentation, complex logic explanation, and TODO markers
5. THE Constitution_Document SHALL define file organization patterns for both Backend_System and Frontend_System
6. THE Constitution_Document SHALL specify that AI_Agent must follow Go official style guide and Vue style guide

### Requirement 3: 架构约束与设计原则

**User Story:** 作为架构师，我希望 AI 严格遵守项目的架构设计和设计原则，以便维护系统的整体一致性。

#### Acceptance Criteria

1. THE Constitution_Document SHALL define the three-layer architecture constraint for Backend_System with api/, app/, and pkg/ directories
2. THE Constitution_Document SHALL specify that AI_Agent must use Protobuf_Schema for all API definitions
3. THE Constitution_Document SHALL specify that AI_Agent must use Ent_Schema for all database model definitions
4. THE Constitution_Document SHALL require AI_Agent to use Event_Bus for cross-module communication instead of direct dependencies
5. THE Constitution_Document SHALL specify that AI_Agent must use Lua_Engine for extensible business logic
6. THE Constitution_Document SHALL define component-based architecture requirements for Frontend_System
7. THE Constitution_Document SHALL prohibit AI_Agent from violating separation of concerns between layers

### Requirement 4: 防止幻觉机制

**User Story:** 作为项目开发者，我希望建立机制防止 AI 臆造不存在的代码元素，以便确保生成代码的可靠性。

#### Acceptance Criteria

1. THE Constitution_Document SHALL require AI_Agent to verify existence of APIs before referencing them
2. THE Constitution_Document SHALL require AI_Agent to verify existence of functions before calling them
3. THE Constitution_Document SHALL require AI_Agent to verify existence of modules before importing them
4. THE Constitution_Document SHALL require AI_Agent to verify existence of configuration keys before using them
5. THE Constitution_Document SHALL specify that AI_Agent must consult project documentation when uncertain about API existence
6. WHEN AI_Agent cannot verify existence of a code element, THE AI_Agent SHALL explicitly ask for confirmation before proceeding
7. THE Constitution_Document SHALL require AI_Agent to provide source references for any external libraries or APIs used

### Requirement 5: 禁止行为清单

**User Story:** 作为项目管理者，我希望明确列出 AI 禁止执行的操作，以便防止潜在的风险和错误。

#### Acceptance Criteria

1. THE Constitution_Document SHALL define Forbidden_Action list including modifying core architecture without approval
2. THE Constitution_Document SHALL prohibit AI_Agent from deleting existing database migrations
3. THE Constitution_Document SHALL prohibit AI_Agent from modifying Protobuf_Schema without explicit instruction
4. THE Constitution_Document SHALL prohibit AI_Agent from bypassing authentication or authorization mechanisms
5. THE Constitution_Document SHALL prohibit AI_Agent from introducing external dependencies without approval
6. THE Constitution_Document SHALL prohibit AI_Agent from modifying production configuration files
7. THE Constitution_Document SHALL prohibit AI_Agent from generating code that violates security best practices
8. THE Constitution_Document SHALL prohibit AI_Agent from creating duplicate functionality that already exists

### Requirement 6: 任务执行流程与留痕机制

**User Story:** 作为项目审计者，我希望 AI 的每个操作都有完整的记录，以便追溯和审查 AI 的决策过程。

#### Acceptance Criteria

1. WHEN AI_Agent starts a task, THE AI_Agent SHALL create a Task_Trace record with task description and timestamp
2. WHEN AI_Agent makes a decision, THE AI_Agent SHALL record the decision rationale in Task_Trace
3. WHEN AI_Agent generates code, THE AI_Agent SHALL record the generated files and modification summary in Task_Trace
4. WHEN AI_Agent completes a task, THE AI_Agent SHALL record the completion status and verification results in Task_Trace
5. THE Constitution_Document SHALL specify the Task_Trace format including required fields and structure
6. THE Constitution_Document SHALL require AI_Agent to maintain Task_Trace in a designated directory
7. THE Task_Trace SHALL include references to relevant documentation or examples used

### Requirement 7: 代码验证要求

**User Story:** 作为质量保证工程师，我希望 AI 生成的代码经过自动验证，以便及早发现问题。

#### Acceptance Criteria

1. WHEN AI_Agent generates Go code, THE AI_Agent SHALL run Code_Validator to check syntax and formatting
2. WHEN AI_Agent generates Vue code, THE AI_Agent SHALL run Code_Validator to check syntax and linting rules
3. WHEN AI_Agent modifies Protobuf_Schema, THE AI_Agent SHALL verify the schema compiles successfully
4. WHEN AI_Agent modifies Ent_Schema, THE AI_Agent SHALL verify the schema generates without errors
5. THE Constitution_Document SHALL require AI_Agent to run existing tests after code modifications
6. IF Code_Validator detects errors, THEN THE AI_Agent SHALL fix the errors before marking task as complete
7. THE Constitution_Document SHALL specify that AI_Agent must verify imports and dependencies are available

### Requirement 8: 错误处理与回滚机制

**User Story:** 作为项目开发者，我希望当 AI 生成的代码出现问题时能够快速回滚，以便保护代码库的稳定性。

#### Acceptance Criteria

1. WHEN AI_Agent detects generated code violates Constitution_Document rules, THE AI_Agent SHALL trigger Rollback_Mechanism
2. WHEN Code_Validator reports critical errors, THE AI_Agent SHALL trigger Rollback_Mechanism
3. THE Rollback_Mechanism SHALL restore all modified files to their previous state
4. THE Rollback_Mechanism SHALL record the rollback action and reason in Task_Trace
5. WHEN Rollback_Mechanism is triggered, THE AI_Agent SHALL report the issue to the developer with detailed explanation
6. THE Constitution_Document SHALL define criteria for when Rollback_Mechanism should be triggered
7. THE Constitution_Document SHALL require AI_Agent to create backup before making significant changes

### Requirement 9: 技术栈特定规范 - Go 后端

**User Story:** 作为后端开发者，我希望 AI 遵循 Go 语言和项目后端技术栈的最佳实践，以便生成高质量的后端代码。

#### Acceptance Criteria

1. THE Constitution_Document SHALL require AI_Agent to use Go modules for dependency management
2. THE Constitution_Document SHALL require AI_Agent to follow Go error handling patterns with explicit error returns
3. THE Constitution_Document SHALL require AI_Agent to use context.Context for request-scoped values and cancellation
4. THE Constitution_Document SHALL require AI_Agent to implement gRPC services according to Protobuf_Schema definitions
5. THE Constitution_Document SHALL require AI_Agent to use Ent_Schema query builders instead of raw SQL
6. THE Constitution_Document SHALL require AI_Agent to publish events to Event_Bus for cross-module notifications
7. THE Constitution_Document SHALL specify concurrency patterns including proper use of goroutines and channels
8. THE Constitution_Document SHALL require AI_Agent to follow Go testing conventions with _test.go files

### Requirement 10: 技术栈特定规范 - Vue 前端

**User Story:** 作为前端开发者，我希望 AI 遵循 Vue 框架和项目前端技术栈的最佳实践，以便生成高质量的前端代码。

#### Acceptance Criteria

1. THE Constitution_Document SHALL require AI_Agent to use Vue 3 Composition API for new components
2. THE Constitution_Document SHALL require AI_Agent to follow Vue component naming conventions with PascalCase
3. THE Constitution_Document SHALL require AI_Agent to properly define component props with type validation
4. THE Constitution_Document SHALL require AI_Agent to use Vue Router for navigation
5. THE Constitution_Document SHALL require AI_Agent to use Pinia or Vuex for state management
6. THE Constitution_Document SHALL require AI_Agent to implement proper component lifecycle management
7. THE Constitution_Document SHALL require AI_Agent to follow Vue template syntax and directives correctly
8. THE Constitution_Document SHALL require AI_Agent to implement proper event handling and emit patterns

### Requirement 11: 文档生成与维护

**User Story:** 作为项目维护者，我希望 AI 在生成代码的同时更新相关文档，以便保持文档与代码的同步。

#### Acceptance Criteria

1. WHEN AI_Agent creates new API endpoints, THE AI_Agent SHALL update API documentation
2. WHEN AI_Agent modifies Protobuf_Schema, THE AI_Agent SHALL update the API reference documentation
3. WHEN AI_Agent adds new components, THE AI_Agent SHALL create component usage documentation
4. WHEN AI_Agent implements new features, THE AI_Agent SHALL update the feature documentation
5. THE Constitution_Document SHALL specify documentation format and location requirements
6. THE Constitution_Document SHALL require AI_Agent to include code examples in documentation
7. THE Constitution_Document SHALL require AI_Agent to document configuration options and environment variables

### Requirement 12: 安全与性能规范

**User Story:** 作为安全工程师，我希望 AI 生成的代码遵循安全和性能最佳实践，以便保护系统安全和性能。

#### Acceptance Criteria

1. THE Constitution_Document SHALL require AI_Agent to validate all user inputs before processing
2. THE Constitution_Document SHALL require AI_Agent to use parameterized queries to prevent SQL injection
3. THE Constitution_Document SHALL require AI_Agent to implement proper authentication checks for protected endpoints
4. THE Constitution_Document SHALL require AI_Agent to implement proper authorization checks based on user roles
5. THE Constitution_Document SHALL require AI_Agent to avoid exposing sensitive information in logs or error messages
6. THE Constitution_Document SHALL require AI_Agent to implement rate limiting for public APIs
7. THE Constitution_Document SHALL require AI_Agent to use efficient database queries with proper indexing considerations
8. THE Constitution_Document SHALL require AI_Agent to implement proper resource cleanup and connection pooling
