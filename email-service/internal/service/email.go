package service

import (
	"context"
	"log"
	"net/smtp"
	"os"

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
	smtpServer := os.Getenv("SMPT_SERVER")
	port := os.Getenv("SMPT_PORT")
	username := os.Getenv("USER_SMPT_MAIL") // ваш email
	password := os.Getenv("USER_SMPT_PASS") // пароль приложения (для двухфакторной аутентификации)
	from := os.Getenv("USER_SMPT_MAIL")     // ваш email

	// Получатель и содержимое письма
	to := []string{req.Email} // получатель из запроса
	subject := req.Message    // тема письма
	body := req.Body          // тело письма

	// Настроить аутентификацию
	auth := smtp.PlainAuth("", username, password, smtpServer)

	// Формирование письма
	message := []byte("Subject: " + subject + "\n\n" + body)

	// Отправка письма в отдельной горутине
	go func() {
		err := smtp.SendMail(smtpServer+":"+port, auth, from, to, message)
		if err != nil {
			log.Printf("Ошибка при отправке письма: %v", err)
		}
	}()
	// Cинхронная отправка
	/*err := smtp.SendMail(smtpServer+":"+port, auth, from, to, message)
	if err != nil {
		log.Printf("Ошибка при отправке письма: %v", err)
	}*/

	// Ответ сразу, без ожидания отправки
	return &pb.SendEmailResponse{
		Message: "Сообщение принято в обработку на отправку",
	}, nil
}
