package utils

import (
	"crmSystem/proto/logs"
	"fmt"
	"golang.org/x/net/context"
)

func SaveLogsError(ctx context.Context, clientLogs logs.LogsServiceClient, database string, userId string, errSave string) error {
	// Формируем лог-запрос
	logRequest := &logs.LogRequest{
		Name:     "auth",
		Level:    "error",
		Message:  fmt.Sprintf("Ошибка: %v", errSave),
		Database: database,
		UserID:   userId,
	}

	//Вызываем сохранение логов
	_, err := clientLogs.SaveLogs(ctx, logRequest)
	if err != nil {
		return err
	}

	return nil
}
