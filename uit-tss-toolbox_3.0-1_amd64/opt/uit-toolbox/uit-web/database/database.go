package database

import (
	"bytes"
	"context"
	"crypto"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/types"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

var csvHeader = []string{
	"Tagnumber",
	"System Serial",
	"Location",
	"Manufacturer",
	"Model",
	"Department",
	"Domain",
	"OS Name",
	"Status",
	"Broken",
	"Note",
	"Last Updated",
}

func VerifyRowsAffected(result sql.Result, expectedRowCount int64) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected != expectedRowCount {
		return fmt.Errorf("unexpected number of rows affected: %d, expected exactly %d row(s)", rowsAffected, expectedRowCount)
	}
	return nil
}

func ToStringPtr(v sql.NullString) *string {
	if v.Valid {
		return &v.String
	}
	return nil
}
func ToInt64Ptr(v sql.NullInt64) *int64 {
	if v.Valid {
		return &v.Int64
	}
	return nil
}
func ToBoolPtr(v sql.NullBool) *bool {
	if v.Valid {
		return &v.Bool
	}
	return nil
}
func ToTimePtr(v sql.NullTime) *time.Time {
	if v.Valid {
		return &v.Time
	}
	return nil
}

func ptrToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}
func ptrToNullInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
func ptrToNullFloat64(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}
func ptrToNullBool(p *bool) sql.NullBool {
	if p == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *p, Valid: true}
}
func ptrToNullTime(p *time.Time) sql.NullTime {
	if p == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *p, Valid: true}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}

func toNullFloat64(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}

func toNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func toNullDuration(d time.Duration) sql.NullInt64 {
	if d == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(d), Valid: true}
}

func ptrIntToString(p *int64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatInt(*p, 10)
}

func ptrStringToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptrBoolToString(p *bool) string {
	if p == nil {
		return ""
	}
	return strconv.FormatBool(*p)
}

func ptrTimeToString(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func CreateAdminUser() error {
	db, err := config.GetDatabaseConn()
	if err != nil {
		return fmt.Errorf("error getting database connection in CreateAdminUser: %w", err)
	}
	adminUsername, adminPasswd, err := config.GetAdminCredentials()
	if err != nil {
		return fmt.Errorf("failed to get admin credentials: %w", err)
	}

	if strings.TrimSpace(adminUsername) == "" {
		return fmt.Errorf("admin username is empty")
	}
	usernameHash := crypto.SHA256.New()
	usernameHash.Write([]byte(adminUsername))
	adminUsernameHash := hex.EncodeToString(usernameHash.Sum(nil))

	if strings.TrimSpace(adminPasswd) == "" {
		return fmt.Errorf("admin password is empty")
	}
	passwordHash := crypto.SHA256.New()
	passwordHash.Write([]byte(adminPasswd))
	adminPasswdHash := hex.EncodeToString(passwordHash.Sum(nil))

	bcryptHashBytes, err := bcrypt.GenerateFromPassword([]byte(adminPasswdHash), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}
	bcryptHashString := string(bcryptHashBytes)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Delete and recreate admin user in table logins
	sqlCode := `
    INSERT INTO logins (username, password, common_name, is_admin, enabled)
    VALUES ($1, $2, 'admin', TRUE, TRUE)
    ON CONFLICT (username)
    DO UPDATE SET 
		username = EXCLUDED.username,
		common_name = EXCLUDED.common_name,
		is_admin = EXCLUDED.is_admin,
		enabled = EXCLUDED.enabled,
		password = EXCLUDED.password;
    `

	_, err = db.ExecContext(ctx, sqlCode, adminUsernameHash, bcryptHashString)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	return nil
}

func ConvertInventoryTableDataToCSV(ctx context.Context, dbQueryData []types.InventoryTableRow) (*bytes.Buffer, error) {
	if len(dbQueryData) == 0 {
		return nil, fmt.Errorf("dbQueryData is nil in ConvertInventoryTableDataToCSV")
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("Context error in ConvertInventoryTableDataToCSV: %w", ctx.Err())
	}

	var buf bytes.Buffer
	buf.Grow(len(dbQueryData) * 200) // Grow by 200 bytes before another allocation
	csvWriter := csv.NewWriter(&buf)

	if err := csvWriter.Write(csvHeader); err != nil {
		return nil, fmt.Errorf("Error writing CSV header in ConvertInventoryTableDataToCSV: %w", err)
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, fmt.Errorf("Error flushing CSV writer after writing header in ConvertInventoryTableDataToCSV: %w", err)
	}

	for _, row := range dbQueryData {
		record := []string{
			ptrIntToString(row.Tagnumber),
			ptrStringToString(row.SystemSerial),
			ptrStringToString(row.LocationFormatted),
			ptrStringToString(row.SystemManufacturer),
			ptrStringToString(row.SystemModel),
			ptrStringToString(row.DepartmentFormatted),
			ptrStringToString(row.DomainFormatted),
			ptrStringToString(row.OsName),
			ptrStringToString(row.Status),
			ptrBoolToString(row.Broken),
			ptrStringToString(row.Note),
			ptrTimeToString(row.LastUpdated),
		}
		if err := csvWriter.Write(record); err != nil {
			return nil, fmt.Errorf("Error writing CSV row in ConvertInventoryTableDataToCSV: %w", err)
		}
	}

	// Flush buffered data to the writer
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, fmt.Errorf("Error flushing CSV writer in ConvertInventoryTableDataToCSV: %w", err)
	}

	return &buf, nil
}
