package utils

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// ConvertPgNumericToDecimal converts pgtype.Numeric to decimal.Decimal
func ConvertPgNumericToDecimal(pgNum pgtype.Numeric) (decimal.Decimal, error) {
	if !pgNum.Valid {
		return decimal.Zero, nil
	}

	// Convert pgtype.Numeric to string and then to decimal.Decimal
	numStr := pgNum.Int.String()
	if pgNum.Exp < 0 {
		// Handle decimal places
		exp := int(-pgNum.Exp)
		if len(numStr) <= exp {
			// Pad with zeros if needed
			numStr = "0." + fmt.Sprintf("%0*s", exp, numStr)
		} else {
			// Insert decimal point
			pos := len(numStr) - exp
			numStr = numStr[:pos] + "." + numStr[pos:]
		}
	}

	dec, err := decimal.NewFromString(numStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse numeric value: %w", err)
	}

	return dec, nil
}

// ConvertDecimalToPgNumeric converts decimal.Decimal to pgtype.Numeric
func ConvertDecimalToPgNumeric(dec decimal.Decimal) pgtype.Numeric {
	var pgNum pgtype.Numeric
	err := pgNum.Scan(dec.String())
	if err != nil {
		// Return invalid numeric on error
		return pgtype.Numeric{Valid: false}
	}
	return pgNum
}

// ConvertPgTextToString converts pgtype.Text to string
func ConvertPgTextToString(pgText pgtype.Text) string {
	if !pgText.Valid {
		return ""
	}
	return pgText.String
}

// ConvertStringToPgText converts string to pgtype.Text
func ConvertStringToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// ConvertPgTimestampToTime converts pgtype.Timestamp to time.Time
func ConvertPgTimestampToTime(pgTime pgtype.Timestamp) time.Time {
	if !pgTime.Valid {
		return time.Time{}
	}
	return pgTime.Time
}

// ConvertTimeToPgTimestamp converts time.Time to pgtype.Timestamp
func ConvertTimeToPgTimestamp(t time.Time) pgtype.Timestamp {
	if t.IsZero() {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: t, Valid: true}
}