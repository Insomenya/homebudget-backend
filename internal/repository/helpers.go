package repository

import "time"

type scannable interface {
	Scan(dest ...any) error
}

func ts() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}