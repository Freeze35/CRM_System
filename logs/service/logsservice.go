package logsservice

import (
	"bytes"
	"context"
	pb "crmSystem/proto/logs"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net/http"
	"time"
)

// LogsServer реализует gRPC сервис для сохранения логов в Loki
type LogsServer struct {
	pb.UnimplementedLogsServiceServer
	lokiURL string
	client  *http.Client
}

// Структура запроса для Loki
type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

type lokiPayload struct {
	Streams []lokiStream `json:"streams"`
}

// NewGRPCDBLogsService создает новый инстанс gRPC сервиса логов
func NewGRPCDBLogsService(lokiURL string) *LogsServer {
	return &LogsServer{
		lokiURL: lokiURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// SaveLogs обрабатывает запросы на сохранение логов
func (s *LogsServer) SaveLogs(_ context.Context, req *pb.LogRequest) (*pb.LogResponse, error) {
	// Создание payload для Loki
	payload := lokiPayload{
		Streams: []lokiStream{
			{
				Stream: map[string]string{
					"job":      req.Name,
					"level":    req.Level,
					"database": req.Database,
					"userId":   req.UserID,
				},
				Values: [][2]string{
					{fmt.Sprintf("%d", time.Now().UnixNano()), req.Message},
				},
			},
		},
	}

	// Кодирование JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка кодирования JSON: %v", err)
	}

	// Отправка запроса в Loki
	url := fmt.Sprintf("%s/loki/api/v1/push", s.lokiURL)
	reqHTTP, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка создания запроса: %v", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(reqHTTP)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка отправки запроса: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Ошибка закрытия соединения")
		}
	}(resp.Body)

	// Проверка статуса ответа
	if resp.StatusCode != http.StatusNoContent {
		return nil, status.Errorf(codes.Internal, "Ошибка Loki: статус %d", resp.StatusCode)
	}

	// Успешный ответ
	return &pb.LogResponse{Message: "Лог сохранен."}, nil
}
