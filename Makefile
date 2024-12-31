# Сборка и запуск контейнеров
up:
	docker-compose up --build

# Остановка контейнеров
down:
	docker-compose down

# Пересобрать контейнеры
rebuild:
	docker-compose down
	docker-compose up --build

# Просмотр логов
logs:
	docker-compose logs -f

# Удалить все контейнеры и данные
clean:
	docker-compose down -v


proto-auth:
	protoc --go_out=./auth/proto --go-grpc_out=./auth/proto ./auth/proto/auth.proto
	protoc --go_out=./auth/proto --go-grpc_out=./auth/proto ./dbservice/proto/dbservice.proto


proto-dbservice:
	protoc --go_out=./dbservice/proto --go-grpc_out=./dbservice/proto ./dbservice/proto/dbservice.proto
	protoc --go_out=./auth/proto --go-grpc_out=./auth/proto ./dbservice/proto/dbservice.proto
	protoc --go_out=./redis/proto --go-grpc_out=./redis/proto ./dbservice/proto/dbservice.proto
	protoc --go_out=./timer/proto --go-grpc_out=./timer/proto ./dbservice/proto/dbservice.proto

proto-redis:
	protoc --go_out=./redis/proto --go-grpc_out=./redis/proto ./redis/proto/redis_service.proto
	protoc --go_out=./dbservice/proto --go-grpc_out=./dbservice/proto ./redis/proto/redis_service.proto

proto-chats:
	protoc --go_out=./chats/proto --go-grpc_out=./chats/proto ./chats/proto/chat.proto
	protoc --go_out=./chats/proto --go-grpc_out=./chats/proto ./dbservice/proto/dbservice.proto

proto-timer:
	protoc --go_out=./timer/proto --go-grpc_out=./timer/proto ./timer/proto/timer.proto
	protoc --go_out=./dbservice/proto --go-grpc_out=./dbservice/proto ./timer/proto/timer.proto

proto-email-service:
	protoc --go_out=./email-service/proto --go-grpc_out=./email-service/proto ./email-service/proto/email.proto
	protoc --go_out=./auth/proto --go-grpc_out=./auth/proto ./email-service/proto/email.proto

#Создать ключи для jwt token (при генерации публичного потребуется ввести пароль создания )
opensslkeys:
	openssl genpkey -algorithm RSA -out ./auth/opensslkeys/private_key.pem -aes256 -pass pass:standard_password
	openssl rsa -in ./auth/opensslkeys/private_key.pem -pubout -out ./auth/opensslkeys/public_key.pem
	copy .\auth\opensslkeys\private_key.pem .\nginx\opensslkeys\private_key.pem
	copy .\auth\opensslkeys\public_key.pem .\nginx\opensslkeys\public_key.pem

auth-ssl:
	openssl req -x509 -newkey rsa:4096 -keyout ./auth/ssl/key.pem -out ./auth/ssl/cert.pem -days 365 -nodes

# Создание ключа для CA
1_ca-key:
	openssl genpkey -algorithm RSA -out ./rootca/ca.key -aes256 -pass pass:standard_password

# Создание корневого сертификата CA
2_cert-ca:
	openssl req -x509 -new -nodes -key ./rootca/ca.key -sha256 -days 3650 -out ./rootca/ca.crt -subj "/CN=crmsystem.com"

# Создание ключа для сервера
3_server-sslkey:
	openssl genpkey -algorithm RSA -out ./auth/sslkeys/server.key
	openssl genpkey -algorithm RSA -out ./dbservice/sslkeys/server.key
	openssl genpkey -algorithm RSA -out ./nginx/sslkeys/server.key
	openssl genpkey -algorithm RSA -out ./email-service/sslkeys/server.key

# Создание запроса на сертификат для сервера
4_server-sslcert:
	openssl req -new -key ./auth/sslkeys/server.key -out ./auth/sslkeys/server.csr -config ./rootca/ssl.cnf
	openssl req -new -key ./dbservice/sslkeys/server.key -out ./dbservice/sslkeys/server.csr -config ./rootca/ssl.cnf
	openssl req -new -key ./nginx/sslkeys/server.key -out ./nginx/sslkeys/server.csr -config ./rootca/ssl.cnf
	openssl req -new -key ./email-service/sslkeys/server.key -out ./email-service/sslkeys/server.csr -config ./rootca/ssl.cnf

# Подпись сертификата сервера через CA
5_trusted-sslcacert:
	openssl x509 -req -in ./auth/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./auth/sslkeys/server.pem -days 3650 -sha256 -extfile ./rootca/ssl.cnf -extensions req_ext
	openssl x509 -req -in ./dbservice/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./dbservice/sslkeys/server.pem -days 3650 -sha256 -extfile ./rootca/ssl.cnf -extensions req_ext
	openssl x509 -req -in ./nginx/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./nginx/sslkeys/server.pem -days 3650 -sha256 -extfile ./rootca/ssl.cnf -extensions req_ext
	openssl x509 -req -in ./email-service/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./email-service/sslkeys/server.pem -days 3650 -sha256 -extfile ./rootca/ssl.cnf -extensions req_ext
