package dbadminservice

import (
	pb "crmSystem/proto/logs"
	"fmt"
	"github.com/grafana/loki-client-go/loki"
	"github.com/prometheus/common/model"
	"time"
)

type LogsServer struct {
	pb.UnimplementedLogsServiceServer
	lokiClient *loki.Client
}

func NewGRPCDBLogsService(lokiClient *loki.Client) *LogsServer {
	return &LogsServer{
		lokiClient: lokiClient,
	}
}

func (s LogsServer) SaveLogs(req *pb.HelloRequest) (*pb.HelloResponse, error) {
	// Логирование в Loki
	labels := model.LabelSet{
		model.LabelName("job"):     model.LabelValue("go-microservice"),
		model.LabelName("level"):   model.LabelValue("info"),
		model.LabelName("handler"): model.LabelValue("say_hello"),
	}
	logMsg := fmt.Sprintf("Получен gRPC запрос от %s", req.GetName())

	err := s.lokiClient.Handle(labels, time.Now(), logMsg)
	if err != nil {
		return nil, err
	}

	// Ответ клиенту
	resp := &pb.HelloResponse{
		Message: "Лог сохранён.",
	}
	return resp, nil
}
