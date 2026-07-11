package health

import "github.com/tapclap/mysql-master-health-checker/internal/masterenv"

// Evaluation is the combined health decision used by HTTP handlers and metrics.
type Evaluation struct {
	Healthy       bool
	Reason        string
	FailureReason FailureReason
	MySQL         Result
	Master        bool
	MySQLUp       bool
	ReadOnly      bool
	ReadOnlyKnown bool
}

// Evaluate combines cached MySQL state with the current MYSQL_MASTER flag.
func Evaluate(store *Store) Evaluation {
	mysql := store.Snapshot()
	master := masterenv.Enabled()

	eval := Evaluation{
		MySQL:         mysql,
		Master:        master,
		MySQLUp:       mysql.Available,
		ReadOnly:      mysql.ReadOnly,
		ReadOnlyKnown: mysql.ReadOnlyKnown,
	}

	switch {
	case !master:
		eval.Healthy = false
		eval.Reason = "MYSQL_MASTER is not enabled"
		eval.FailureReason = FailureReasonMasterEnv
	case !mysql.Available:
		eval.Healthy = false
		eval.Reason = "mysql is unavailable"
		eval.FailureReason = FailureReasonMySQLUnavailable
	case mysql.ReadOnlyKnown && mysql.ReadOnly:
		eval.Healthy = false
		eval.Reason = "mysql read_only is enabled"
		eval.FailureReason = FailureReasonReadOnly
	case !mysql.ReadOnlyKnown:
		eval.Healthy = false
		eval.Reason = "mysql read_only state is unknown"
		eval.FailureReason = FailureReasonMySQLUnavailable
	default:
		eval.Healthy = true
		eval.Reason = "OK"
		eval.FailureReason = FailureReasonNone
	}

	return eval
}
