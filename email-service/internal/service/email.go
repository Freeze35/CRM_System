package service

import (
	"context"
	"net/smtp"

	pb "crmSystem/proto/email-service"
)

type EmailService struct {
	pb.UnimplementedEmailServiceServer
}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func (s *EmailService) SendEmail(_ context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	// Параметры SMTP сервера Gmail
	smtpServer := "smtp.gmail.com"
	port := "587"
	username := "your-email@gmail.com" // ваш email
	password := "your-app-password"    // пароль приложения (для двухфакторной аутентификации)
	from := "your-email@gmail.com"     // ваш email

	// Получатель и содержимое письма
	to := []string{req.Recipient} // получатель из запроса
	subject := req.Subject        // тема письма
	body := req.Body              // тело письма

	// Настроить аутентификацию
	auth := smtp.PlainAuth("", username, password, smtpServer)

	// Формирование письма
	message := []byte("Subject: " + subject + "\n\n" + body)

	// Отправка письма
	err := smtp.SendMail(smtpServer+":"+port, auth, from, to, message)
	if err != nil {
		return &pb.SendEmailResponse{
			Status:  "FAILED",
			Message: err.Error(),
		}, nil
	}

	return &pb.SendEmailResponse{
		Status:  "SUCCESS",
		Message: "Email sent successfully",
	}, nil
}
