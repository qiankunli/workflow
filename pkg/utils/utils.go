package utils

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ToStringP ...
func ToStringP(str string) *string {
	return &str
}

// ToNullableStringP ...
func ToNullableStringP(str string) *string {
	if "" == str {
		return nil
	}
	return &str
}

// FirstNotNullString ...
func FirstNotNullString(first, second string) string {
	if len(first) > 0 {
		return first
	}
	return second
}

func FirstNotNull(first, second interface{}) interface{} {
	if first != nil {
		if firstString, ok := first.(string); ok {
			if len(firstString) == 0 {
				return second
			}
		}
		return first
	}
	return second
}

// ToInt ...
func ToInt(s string, defaultValue int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

// ToInt32 ...
func ToInt32(s string, defaultValue int32) int32 {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(i)
}

func ToInt64(s string, defaultValue int64) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultValue
	}
	return int64(i)
}

// ToUInt ...
func ToUInt(s string, defaultValue uint) uint {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultValue
	}
	return uint(i)
}

// ToInt32P ...
func ToInt32P(i int32) *int32 {
	return &i
}

// ToInt64P ...
func ToInt64P(i int64) *int64 {
	return &i
}

// ToFloat64P ...
func ToFloat64P(f float64) *float64 {
	return &f
}

func GetFilesWithRelPath(toWalkPath string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(toWalkPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(toWalkPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func GenUserName(account int64, user *int64) string {
	if user == nil {
		return strconv.FormatInt(account, 10)
	}
	return fmt.Sprintf("%du%d", account, *user)
}

// EncodeResourceName return an encoded field with len 56
func EncodeResourceName(name string) string {
	hash := sha256.Sum256([]byte(name))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return encoded
}

// ConvertTimePtrToString converts the give time ptr to a RFC3339-compliant string
func ConvertTimePtrToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return ConvertTimeToString(*t)
}

// ConvertTimeToString converts the given time to a RFC3339-compliant string
func ConvertTimeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// HashSha256 ...
func HashSha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}
