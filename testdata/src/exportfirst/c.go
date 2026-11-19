package exportfirst

var Default = New()

var _ = helper

var (
	errClosed = 1
	ErrFull   = 2 // want `exported ErrFull is declared below unexported errClosed \(line 8\); move exported vars above unexported ones`
)

var Limit = 3 // want `exported Limit is declared below unexported errClosed \(line 8\)`
