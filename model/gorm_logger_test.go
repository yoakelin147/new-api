package model

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSanitizeDBErrorRemovesDriverDetails(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"mysql", &mysql.MySQLError{Number: 1062, Message: "duplicate secret-value"}, "mysql error 1062"},
		{"postgres", &pgconn.PgError{Code: "23505", Detail: "secret-value"}, "postgres error SQLSTATE 23505"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sanitizeDBError(test.err)
			require.Equal(t, test.want, got.Error())
			require.NotContains(t, got.Error(), "secret-value")
		})
	}
}

func TestGormLoggerParameterizesSQLiteErrors(t *testing.T) {
	previousDebug := common.DebugEnabled
	t.Cleanup(func() { common.DebugEnabled = previousDebug })
	common.DebugEnabled = false
	var output bytes.Buffer
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: newGormLogger(&output)})
	require.NoError(t, err)
	db.Exec("SELECT * FROM missing_table WHERE token = ?", "secret-value")
	text := output.String()
	require.Contains(t, text, "sqlite error")
	require.NotContains(t, text, "secret-value")
}

func TestSanitizeDBErrorKeepsNonDriverErrors(t *testing.T) {
	err := fmt.Errorf("dial tcp 127.0.0.1:3306: connect: connection refused")
	require.Equal(t, err, sanitizeDBError(err))
}
