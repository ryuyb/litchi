package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents a structured error code in the Litchi system.
type ErrorCode struct {
	Code     string // e.g., L1SYS0001
	Message  string // Human-readable message
	Category string // SYS, AGE, GIT, NET, ENV
	Severity int    // 1=Critical, 2=High, 3=Medium, 4=Low
}

// Error represents a structured error with code and context.
type Error struct {
	Code    ErrorCode
	Detail  string
	Context map[string]interface{}
	Cause   error // Original error if wrapping
}

func (e *Error) Error() string {
	if e.Detail != "" && e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (cause: %v)", e.Code.Code, e.Code.Message, e.Detail, e.Cause)
	}
	if e.Detail != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code.Code, e.Code.Message, e.Detail)
	}
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s (cause: %v)", e.Code.Code, e.Code.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code.Code, e.Code.Message)
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// New creates a new Error with the given code.
func New(code ErrorCode) *Error {
	return &Error{Code: code}
}

// WithDetail adds detail to the error.
func (e *Error) WithDetail(detail string) *Error {
	e.Detail = detail
	return e
}

// WithContext adds context to the error.
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Wrap wraps an existing error with an error code.
func Wrap(code ErrorCode, err error) *Error {
	return &Error{
		Code:  code,
		Cause: err,
	}
}

// WrapWithDetail wraps an existing error with an error code and detail.
func WrapWithDetail(code ErrorCode, err error, detail string) *Error {
	return &Error{
		Code:   code,
		Cause:  err,
		Detail: detail,
	}
}

// Is checks if an error matches a specific error code.
// Supports Go 1.13+ error chain via errors.As.
func Is(err error, code ErrorCode) bool {
	var litchiErr *Error
	if errors.As(err, &litchiErr) {
		return litchiErr.Code.Code == code.Code
	}
	return false
}

// GetCode extracts the error code from an error.
// Uses errors.As to support error chain traversal.
func GetCode(err error) string {
	var litchiErr *Error
	if errors.As(err, &litchiErr) {
		return litchiErr.Code.Code
	}
	return "UNKNOWN"
}

// GetSeverity extracts the severity level from an error.
// Uses errors.As to support error chain traversal.
func GetSeverity(err error) int {
	var litchiErr *Error
	if errors.As(err, &litchiErr) {
		return litchiErr.Code.Severity
	}
	return 0
}

// Predefined error codes organized by category and severity.
// Format: L{Severity}{Category}{Number}
// Severity: 1=Critical, 2=High, 3=Medium, 4=Low
// Category: SYS=System, AGE=Agent, GIT=GitHub, NET=Network, ENV=Environment, API=API, DOM=Domain

// Critical errors (Severity 1) - System level
var (
	ErrDatabaseConnection = ErrorCode{Code: "L1SYS0001", Message: "Database connection failed", Category: "SYS", Severity: 1}
	ErrDatabaseOperation  = ErrorCode{Code: "L1SYS0005", Message: "Database operation failed", Category: "SYS", Severity: 1}
	ErrConfigLoadFailed   = ErrorCode{Code: "L1SYS0002", Message: "Configuration load failed", Category: "SYS", Severity: 1}
	ErrServerStartFailed  = ErrorCode{Code: "L1SYS0003", Message: "Server startup failed", Category: "SYS", Severity: 1}
	ErrMigrationFailed    = ErrorCode{Code: "L1SYS0004", Message: "Database migration failed", Category: "SYS", Severity: 1}
)

// High severity errors (Severity 2) - Agent level
var (
	ErrAgentProcessCrash     = ErrorCode{Code: "L2AGE0001", Message: "Agent process crashed", Category: "AGE", Severity: 2}
	ErrAgentContextLost      = ErrorCode{Code: "L2AGE0002", Message: "Agent session context lost", Category: "AGE", Severity: 2}
	ErrAgentExecutionFail    = ErrorCode{Code: "L2AGE0003", Message: "Agent execution failed", Category: "AGE", Severity: 2}
	ErrAgentTimeout          = ErrorCode{Code: "L2AGE0004", Message: "Agent execution timeout", Category: "AGE", Severity: 2}
	ErrAgentPermissionDenied = ErrorCode{Code: "L2AGE0005", Message: "Agent permission denied", Category: "AGE", Severity: 2}
)

// Medium severity errors (Severity 3) - External services
var (
	ErrGitHubAPIRateLimit = ErrorCode{Code: "L3GIT0001", Message: "GitHub API rate limit exceeded", Category: "GIT", Severity: 3}
	ErrGitHubAuthFailed   = ErrorCode{Code: "L3GIT0002", Message: "GitHub authentication failed", Category: "GIT", Severity: 3}
	ErrGitHubAPIError     = ErrorCode{Code: "L3GIT0003", Message: "GitHub API error", Category: "GIT", Severity: 3}
	ErrWebhookInvalidSig  = ErrorCode{Code: "L3GIT0004", Message: "Webhook signature verification failed", Category: "GIT", Severity: 3}

	// Git branch operation errors (L3GIT0005-L3GIT0010)
	ErrGitBranchCreateFailed = ErrorCode{Code: "L3GIT0005", Message: "Git branch creation failed", Category: "GIT", Severity: 3}
	ErrGitBranchNotFound     = ErrorCode{Code: "L3GIT0006", Message: "Git branch not found", Category: "GIT", Severity: 3}
	ErrGitBranchDeleteFailed = ErrorCode{Code: "L3GIT0007", Message: "Git branch deletion failed", Category: "GIT", Severity: 3}
	ErrGitBranchSwitchFailed = ErrorCode{Code: "L3GIT0008", Message: "Git branch switch failed", Category: "GIT", Severity: 3}
	ErrGitBranchNameInvalid  = ErrorCode{Code: "L3GIT0009", Message: "Git branch name invalid", Category: "GIT", Severity: 3}
	ErrGitBranchExists       = ErrorCode{Code: "L3GIT0010", Message: "Git branch already exists", Category: "GIT", Severity: 3}

	// Git worktree operation errors (L3GIT0011-L3GIT0015)
	ErrGitWorktreeCreateFailed = ErrorCode{Code: "L3GIT0011", Message: "Git worktree creation failed", Category: "GIT", Severity: 3}
	ErrGitWorktreeNotFound     = ErrorCode{Code: "L3GIT0012", Message: "Git worktree not found", Category: "GIT", Severity: 3}
	ErrGitWorktreeDeleteFailed = ErrorCode{Code: "L3GIT0013", Message: "Git worktree deletion failed", Category: "GIT", Severity: 3}
	ErrGitWorktreeLocked       = ErrorCode{Code: "L3GIT0014", Message: "Git worktree is locked", Category: "GIT", Severity: 3}
	ErrGitWorktreePathExists   = ErrorCode{Code: "L3GIT0015", Message: "Git worktree path already exists", Category: "GIT", Severity: 3}

	// Git commit operation errors (L3GIT0016-L3GIT0021)
	ErrGitCommitFailed     = ErrorCode{Code: "L3GIT0016", Message: "Git commit failed", Category: "GIT", Severity: 3}
	ErrGitPushFailed       = ErrorCode{Code: "L3GIT0017", Message: "Git push failed", Category: "GIT", Severity: 3}
	ErrGitAddFailed        = ErrorCode{Code: "L3GIT0018", Message: "Git add failed", Category: "GIT", Severity: 3}
	ErrGitNothingToCommit  = ErrorCode{Code: "L3GIT0019", Message: "Git nothing to commit", Category: "GIT", Severity: 3}
	ErrGitMergeConflict    = ErrorCode{Code: "L3GIT0020", Message: "Git merge conflict detected", Category: "GIT", Severity: 3}
	ErrGitConflictDetected = ErrorCode{Code: "L3GIT0021", Message: "Git conflict detected", Category: "GIT", Severity: 3}

	// Git general errors (L3GIT0022-L3GIT0028)
	ErrGitRepoNotFound      = ErrorCode{Code: "L3GIT0022", Message: "Git repository not found", Category: "GIT", Severity: 3}
	ErrGitRepoOpenFailed    = ErrorCode{Code: "L3GIT0023", Message: "Git repository open failed", Category: "GIT", Severity: 3}
	ErrGitCloneFailed       = ErrorCode{Code: "L3GIT0024", Message: "Git clone failed", Category: "GIT", Severity: 3}
	ErrGitFetchFailed       = ErrorCode{Code: "L3GIT0025", Message: "Git fetch failed", Category: "GIT", Severity: 3}
	ErrGitCommandFailed     = ErrorCode{Code: "L3GIT0026", Message: "Git command execution failed", Category: "GIT", Severity: 3}
	ErrGitAuthentication    = ErrorCode{Code: "L3GIT0027", Message: "Git authentication failed", Category: "GIT", Severity: 3}
	ErrGitOperationFailed   = ErrorCode{Code: "L3GIT0028", Message: "Git operation failed", Category: "GIT", Severity: 3}

	ErrNetworkTimeout     = ErrorCode{Code: "L3NET0001", Message: "Network timeout", Category: "NET", Severity: 3}
	ErrNetworkConnection  = ErrorCode{Code: "L3NET0002", Message: "Network connection failed", Category: "NET", Severity: 3}
	ErrTestEnvUnavailable = ErrorCode{Code: "L3ENV0001", Message: "Test environment unavailable", Category: "ENV", Severity: 3}
)

// Low severity errors (Severity 4) - Business logic
var (
	ErrTaskSkipped            = ErrorCode{Code: "L4TASK0001", Message: "Task was skipped", Category: "DOM", Severity: 4}
	ErrNoTestsFound           = ErrorCode{Code: "L4ENV0001", Message: "No tests found", Category: "ENV", Severity: 4}
	ErrTaskAlreadyComplete    = ErrorCode{Code: "L4TASK0002", Message: "Task already completed", Category: "DOM", Severity: 4}
	ErrSessionNotFound        = ErrorCode{Code: "L4DOM0001", Message: "Work session not found", Category: "DOM", Severity: 4}
	ErrIssueNotFound          = ErrorCode{Code: "L4DOM0002", Message: "Issue not found", Category: "DOM", Severity: 4}
	ErrInvalidStage           = ErrorCode{Code: "L4DOM0003", Message: "Invalid stage", Category: "DOM", Severity: 4}
	ErrInvalidStageTransition = ErrorCode{Code: "L4DOM0007", Message: "Invalid stage transition", Category: "DOM", Severity: 4}
	ErrInvalidTaskStatus      = ErrorCode{Code: "L4DOM0004", Message: "Invalid task status", Category: "DOM", Severity: 4}
	ErrInvalidComplexityScore = ErrorCode{Code: "L4DOM0005", Message: "Invalid complexity score", Category: "DOM", Severity: 4}
	ErrInvalidClarityScore    = ErrorCode{Code: "L4DOM0006", Message: "Invalid clarity score", Category: "DOM", Severity: 4}
	ErrVersionConflict        = ErrorCode{Code: "L4DOM0008", Message: "Version conflict (optimistic lock)", Category: "DOM", Severity: 4}
	ErrPermissionDenied       = ErrorCode{Code: "L4API0001", Message: "Permission denied", Category: "API", Severity: 4}
	ErrValidationFailed       = ErrorCode{Code: "L4API0002", Message: "Validation failed", Category: "API", Severity: 4}
	ErrBadRequest             = ErrorCode{Code: "L4API0003", Message: "Bad request", Category: "API", Severity: 4}

	// Git naming convention errors
	ErrGitBranchNamingViolation = ErrorCode{Code: "L4GIT0001", Message: "Git branch naming convention violation", Category: "GIT", Severity: 4}
)

// API response error codes (for HTTP responses)
type APIErrorCode struct {
	Code    int
	Message string
}

var (
	APIErrBadRequest         = APIErrorCode{Code: 400, Message: "Bad request"}
	APIErrUnauthorized       = APIErrorCode{Code: 401, Message: "Unauthorized"}
	APIErrForbidden          = APIErrorCode{Code: 403, Message: "Forbidden"}
	APIErrNotFound           = APIErrorCode{Code: 404, Message: "Not found"}
	APIErrConflict           = APIErrorCode{Code: 409, Message: "Conflict"}
	APIErrInternal           = APIErrorCode{Code: 500, Message: "Internal server error"}
	APIErrServiceUnavailable = APIErrorCode{Code: 503, Message: "Service unavailable"}
)

// ToAPIError converts domain error to API error code.
// Uses errors.As to support error chain traversal.
func ToAPIError(err error) APIErrorCode {
	var litchiErr *Error
	if errors.As(err, &litchiErr) {
		switch litchiErr.Code.Severity {
		case 1:
			return APIErrInternal
		case 2:
			return APIErrInternal
		case 3, 4:
			if litchiErr.Code.Category == "API" {
				if Is(err, ErrPermissionDenied) {
					return APIErrForbidden
				}
				if Is(err, ErrValidationFailed) || Is(err, ErrBadRequest) {
					return APIErrBadRequest
				}
			}
			if Is(err, ErrSessionNotFound) || Is(err, ErrIssueNotFound) {
				return APIErrNotFound
			}
			if Is(err, ErrVersionConflict) {
				return APIErrConflict
			}
			return APIErrBadRequest
		}
	}
	return APIErrInternal
}
