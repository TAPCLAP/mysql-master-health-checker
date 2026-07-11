package health

// FailureReason identifies which check caused an unhealthy response.
type FailureReason string

const (
	FailureReasonNone             FailureReason = "none"
	FailureReasonMasterEnv        FailureReason = "mysql_master_env"
	FailureReasonMySQLUnavailable FailureReason = "mysql_unavailable"
	FailureReasonReadOnly         FailureReason = "mysql_read_only"
)
