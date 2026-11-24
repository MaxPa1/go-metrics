// Package retry реализует повтор операций при временных ошибках.
package retry

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// Delays задаёт паузы между повторными попытками.
var Delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// IsRetriable решает, стоит ли повторять операцию после ошибки.
type IsRetriable func(error) bool

// Do выполняет fn и повторяет её с паузами из Delays, пока ошибка признаётся временной.
func Do(ctx context.Context, isRetriable IsRetriable, fn func() error) error {
	var err error
	for attempt := 0; ; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if attempt >= len(Delays) || !isRetriable(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(Delays[attempt]):
		}
	}
}

var pgConnectionErrorCodes = map[string]struct{}{
	pgerrcode.ConnectionException:                           {},
	pgerrcode.ConnectionDoesNotExist:                        {},
	pgerrcode.ConnectionFailure:                             {},
	pgerrcode.SQLClientUnableToEstablishSQLConnection:       {},
	pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection: {},
	pgerrcode.TransactionResolutionUnknown:                  {},
	pgerrcode.ProtocolViolation:                             {},
}

// IsRetriablePgError сообщает, является ли ошибка временной ошибкой соединения с PostgreSQL.
func IsRetriablePgError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		_, retriable := pgConnectionErrorCodes[pgErr.Code]
		return retriable
	}

	return IsRetriableNetError(err)
}

// IsRetriableNetError сообщает, является ли ошибка временной сетевой ошибкой.
func IsRetriableNetError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var opErr *net.OpError
	return errors.As(err, &opErr)
}
