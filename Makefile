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

proto-chats:
	protoc --go_out=./chats/proto --go-grpc_out=./chats/proto ./chats/proto/chat.proto
	protoc --go_out=./chats/proto --go-grpc_out=./chats/proto ./dbservice/proto/dbservice.proto

#Создать ключи для jwt token (при генерации публичного потребуется ввести пароль создания )
opensslkeys:
	openssl genpkey -algorithm RSA -out ./auth/opensslkeys/private_key.pem -aes256 -pass pass:standard_password
	openssl rsa -in ./auth/opensslkeys/private_key.pem -pubout -out ./auth/opensslkeys/public_key.pem
	copy .\auth\opensslkeys\private_key.pem .\nginx\opensslkeys\private_key.pem
	copy .\auth\opensslkeys\public_key.pem .\nginx\opensslkeys\public_key.pem

auth-ssl:
	openssl req -x509 -newkey rsa:4096 -keyout ./auth/ssl/key.pem -out ./auth/ssl/cert.pem -days 365 -nodes

# Создание ключа для CA
ca-key:
	openssl genpkey -algorithm RSA -out ./auth/sslkeys/ca.key -aes256 -pass pass:standard_password

# Создание корневого сертификата CA
cert-ca:
	openssl req -x509 -new -nodes -key ./auth/sslkeys/ca.key -sha256 -days 3650 -out ./auth/sslkeys/ca.crt -subj "/CN=yourdomain.com"

# Создание ключа для сервера
server-sslkey:
	openssl genpkey -algorithm RSA -out ./auth/sslkeys/server.key
# Создание запроса на сертификат для сервера
server-sslcert:
	openssl req -new -key ./auth/sslkeys/server.key -out ./auth/sslkeys/server.csr -subj "/CN=yourdomain.com"

# Подпись сертификата сервера через CA
trusted-sslcacert:
	openssl x509 -req -in ./auth/sslkeys/server.csr -CA ./auth/sslkeys/ca.crt -CAkey ./auth/sslkeys/ca.key -CAcreateserial -out ./auth/sslkeys/server.pem -days 365 -sha256