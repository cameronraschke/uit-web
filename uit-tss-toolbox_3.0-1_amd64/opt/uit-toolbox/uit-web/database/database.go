package database

import (
	"bytes"
	"context"
	"crypto"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
	"uit-toolbox/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

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

func ToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}
func ToNullInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
func ToNullBool(p *bool) sql.NullBool {
	if p == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *p, Valid: true}
}
func ToNullTime(p *time.Time) sql.NullTime {
	if p == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *p, Valid: true}
}

func CreateAdminUser() error {
	db := config.GetDatabaseConn()
	if db == nil {
		return errors.New("database connection is not initialized")
	}
	adminUsername, adminPasswd, err := config.GetAdminCredentials()
	if err != nil {
		return errors.New("failed to get admin credentials: " + err.Error())
	}
	if adminUsername == nil || adminPasswd == nil {
		return errors.New("admin credentials are nil")
	}

	if strings.TrimSpace(*adminUsername) == "" {
		return errors.New("admin username is empty")
	}
	usernameHash := crypto.SHA256.New()
	usernameHash.Write([]byte(*adminUsername))
	adminUsernameHash := hex.EncodeToString(usernameHash.Sum(nil))

	if strings.TrimSpace(*adminPasswd) == "" {
		return errors.New("admin password is empty")
	}
	passwordHash := crypto.SHA256.New()
	passwordHash.Write([]byte(*adminPasswd))
	adminPasswdHash := hex.EncodeToString(passwordHash.Sum(nil))

	bcryptHashBytes, err := bcrypt.GenerateFromPassword([]byte(adminPasswdHash), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash admin password: " + err.Error())
	}
	bcryptHashString := string(bcryptHashBytes)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Delete and recreate admin user in table logins
	sqlCode := `
    INSERT INTO logins (username, password, common_name, is_admin, enabled)
    VALUES ($1, $2, 'admin', TRUE, TRUE)
    ON CONFLICT (username)
    DO UPDATE SET password = EXCLUDED.password
    `

	_, err = db.ExecContext(ctx, sqlCode, adminUsernameHash, bcryptHashString)
	if err != nil {
		return errors.New("failed to create admin user: " + err.Error())
	}

	return nil
}

func ConvertInventoryTableDataToCSV(ctx context.Context, dbQueryData []*InventoryTableData) (string, error) {
	if ctx.Err() != nil {
		return "", errors.New("Context error in ConvertInventoryTableDataToCSV: " + ctx.Err().Error())
	}
	if dbQueryData == nil {
		return "", errors.New("dbQueryData is nil in ConvertInventoryTableDataToCSV")
	}
	if len(dbQueryData) == 0 {
		return "", errors.New("no data available to convert to CSV in ConvertInventoryTableDataToCSV")
	}
	writer := new(bytes.Buffer)
	csvWriter := csv.NewWriter(writer)

	csvHeader := []string{
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
	if err := csvWriter.Write(csvHeader); err != nil {
		return "", errors.New("Error writing CSV header in ConvertInventoryTableDataToCSV: " + err.Error())
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
			return "", errors.New("Error writing CSV row in ConvertInventoryTableDataToCSV: " + err.Error())
		}
	}

	// Flush buffered data to the writer
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return "", errors.New("Error flushing CSV writer in ConvertInventoryTableDataToCSV: " + err.Error())
	}

	return writer.String(), nil
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

func ptrTimeToString(p *time.Time) string {
	if p == nil {
		return ""
	}
	return p.Format("2006-01-02 15:04:05")
}
