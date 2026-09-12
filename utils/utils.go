package utils

import (
	"cmp"
	"encoding/binary"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	S3_STANDARD_TIME_FORMAT = time.RFC3339
)

func GenerateDeterministicUUID(str string, ns uuid.UUID) string {
	return uuid.NewSHA1(ns, []byte(str)).String()
}

func ConvertTimeToString(t time.Time) string {
	return t.UTC().Format(S3_STANDARD_TIME_FORMAT)
}

func ConvertStringToTime(timeString string) (*time.Time, error) {
	t, err := time.Parse(S3_STANDARD_TIME_FORMAT, timeString)
	if err != nil {
		return nil, err
	}

	t = t.UTC()
	return &t, nil
}

func FormatTimeString(timeString string) (string, error) {
	t, err := ConvertStringToTime(timeString)
	if err != nil {
		return "", err
	}

	return ConvertTimeToString(*t), nil
}

func AtoiDefault(s string, def int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

func Uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func BytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func SearchSorted[T cmp.Ordered](s []T, v T) (int, bool) {
	return slices.BinarySearch(s, v)
}

func InsertSorted[T cmp.Ordered](s []T, v T) []T {
	i, found := SearchSorted(s, v)
	if found {
		return s // value already present
	}

	s = append(s, v)     // grow slice
	copy(s[i+1:], s[i:]) // shift right
	s[i] = v             // insert

	return s
}

func RemoveSorted[T cmp.Ordered](s []T, v T) ([]T, bool) {
	i, found := SearchSorted(s, v)
	if !found {
		return s, false // value not present
	}

	// Remove element at index i
	return append(s[:i], s[i+1:]...), true
}

func RemoveUnordered[T any](s []T, i int) []T {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
