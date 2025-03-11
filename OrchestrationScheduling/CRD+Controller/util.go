package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
)

func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

func toString(obj interface{}) string {
	b, _ := json.Marshal(obj)
	return string(b)
}
