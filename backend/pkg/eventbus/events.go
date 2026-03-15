package eventbus

// Common event types
const (
	// Email events
	EventEmailReceived  = "email.received"
	EventEmailSent      = "email.sent"
	EventEmailDeleted   = "email.deleted"
	EventEmailRead      = "email.read"
	EventEmailFlagged   = "email.flagged"
	EventEmailProcessed = "email.processed"
	EventEmailFailed    = "email.failed"

	// User events
	EventUserCreated   = "user.created"
	EventUserUpdated   = "user.updated"
	EventUserDeleted   = "user.deleted"
	EventUserLoggedIn  = "user.logged_in"
	EventUserLoggedOut = "user.logged_out"

	// Consumer events (C端用户)
	TopicUserRegistered  = "consumer.user.registered"
	TopicPaymentSuccess  = "consumer.payment.success"
	TopicLogisticsStatus = "consumer.logistics.status_changed"

	// Task events
	EventTaskCreated   = "task.created"
	EventTaskStarted   = "task.started"
	EventTaskCompleted = "task.completed"
	EventTaskFailed    = "task.failed"
	EventTaskCancelled = "task.cancelled"

	// System events
	EventSystemStarted = "system.started"
	EventSystemStopped = "system.stopped"
	EventSystemError   = "system.error"
)

// EmailReceivedEvent represents an email received event
type EmailReceivedEvent struct {
	EmailID   string `json:"email_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Mailbox   string `json:"mailbox"`
	AccountID string `json:"account_id"`
	TenantID  uint32 `json:"tenant_id,omitempty"`
}

// EmailProcessedEvent represents an email processed event
type EmailProcessedEvent struct {
	EmailID   string `json:"email_id"`
	AccountID string `json:"account_id"`
	Mailbox   string `json:"mailbox"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// UserCreatedEvent represents a user created event
type UserCreatedEvent struct {
	UserID   uint32 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// TaskCompletedEvent represents a task completed event
type TaskCompletedEvent struct {
	TaskID   string                 `json:"task_id"`
	TaskType string                 `json:"task_type"`
	Success  bool                   `json:"success"`
	Error    string                 `json:"error,omitempty"`
	Result   map[string]interface{} `json:"result,omitempty"`
	Duration int64                  `json:"duration_ms"`
}

// SystemErrorEvent represents a system error event
type SystemErrorEvent struct {
	Component string `json:"component"`
	Error     string `json:"error"`
	Severity  string `json:"severity"`
}

// UserRegisteredEvent C端用户注册事件
type UserRegisteredEvent struct {
	TenantID uint32 `json:"tenant_id"`
	UserID   uint32 `json:"user_id"`
	Phone    string `json:"phone"`
	Nickname string `json:"nickname,omitempty"`
}

// PaymentSuccessEvent 支付成功事件
type PaymentSuccessEvent struct {
	TenantID  uint32 `json:"tenant_id"`
	OrderNo   string `json:"order_no"`
	UserID    uint32 `json:"user_id"`
	Amount    int64  `json:"amount"` // 金额(分)
	PaymentMethod string `json:"payment_method"`
}

// LogisticsStatusChangedEvent 物流状态变更事件
type LogisticsStatusChangedEvent struct {
	TenantID    uint32 `json:"tenant_id"`
	TrackingNo  string `json:"tracking_no"`
	OldStatus   string `json:"old_status"`
	NewStatus   string `json:"new_status"`
	UpdatedAt   int64  `json:"updated_at"`
}
