package utils

import (
	"math/rand"
	"strings"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomDBName(numberSymbols int) string {
	// Создаем новый генератор случайных чисел с seed на основе текущего времени
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sb := strings.Builder{}
	sb.Grow(numberSymbols)
	for i := 0; i < numberSymbols; i++ {
		sb.WriteByte(letterBytes[r.Intn(len(letterBytes))])
	}
	return sb.String()
}
