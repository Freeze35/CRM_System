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

proto-dbservice:
	protoc --go_out=./dbservice/proto --go-grpc_out=./dbservice/proto ./dbservice/proto/dbservice.proto

opensslkeys:
	openssl genpkey -algorithm RSA -out ./nginx/opensslkeys/private_key.pem -aes256 -pass pass:standard_password
	openssl rsa -in ./nginx/opensslkeys/private_key.pem -pubout -out ./nginx/opensslkeys/public_key.pem

auth-ssl:
	openssl req -x509 -newkey rsa:4096 -keyout auth/ssl/key.pem -out auth/ssl/cert.pem -days 365 -nodes

# Создание ключа для CA
ca-key:
	openssl genpkey -algorithm RSA -out sslkeys/ca.key -aes256 -pass pass:standard_password

# Создание корневого сертификата CA
cert-ca:
	openssl req -x509 -new -nodes -key sslkeys/ca.key -sha256 -days 3650 -out sslkeys/ca.crt -subj "/CN=Your Root CA"

# Создание ключа для сервера
server-sslkey:
	openssl genpkey -algorithm RSA -out sslkeys/server.key
# Создание запроса на сертификат для сервера
server-sslcert:
	openssl req -new -key sslkeys/server.key -out sslkeys/server.csr -subj "/CN=yourdomain.com"

# Подпись сертификата сервера через CA
trusted-sslcacert:
	openssl x509 -req -in nginx/sslkeys/server.csr -CA nginx/sslkeys/ca.crt -CAkey nginx/sslkeys/ca.key -CAcreateserial -out nginx/sslkeys/server.pem -days 365 -sha256