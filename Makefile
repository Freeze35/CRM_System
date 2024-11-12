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

1server-key:
	openssl genpkey -algorithm RSA -out ./auth/sslkeys/server.key
	copy .\auth\sslkeys\server.key .\nginx\sslkeys\server.key
	copy .\auth\sslkeys\server.key .\dbservice\sslkeys\server.key

2CSR-sert:
	openssl req -new -key ./auth/sslkeys/server.key -out ./auth/sslkeys/server.csr -config ssl.cnf
	copy .\auth\sslkeys\server.csr .\nginx\sslkeys\server.csr
	copy .\auth\sslkeys\server.csr .\dbservice\sslkeys\server.csr

3root-key:
	openssl genpkey -algorithm RSA -out ./auth/sslkeys/ca.key -aes256 -pass pass:standard_password
	copy .\auth\sslkeys\ca.key .\nginx\sslkeys\ca.key
	copy .\auth\sslkeys\ca.key .\dbservice\sslkeys\ca.key

4self-signed-root-certificate:
	openssl req -x509 -new -nodes -key ./auth/sslkeys/ca.key -sha256 -days 3650 -out ./auth/sslkeys/ca.crt -config ssl.cnf
	copy .\auth\sslkeys\ca.crt .\nginx\sslkeys\ca.crt
	copy .\auth\sslkeys\ca.crt .\dbservice\sslkeys\ca.crt

5Signing-a-server-certificate-via-a-CA:
	openssl x509 -req -in ./auth/sslkeys/server.csr -CA ./auth/sslkeys/ca.crt -CAkey ./auth/sslkeys/ca.key -CAcreateserial -out ./auth/sslkeys/server.pem -days 365 -sha256 -extfile ssl.cnf -extensions req_ext
	copy .\auth\sslkeys\server.pem .\nginx\sslkeys\server.pem
	copy .\auth\sslkeys\server.pem .\dbservice\sslkeys\server.pem
	copy .\auth\sslkeys\ca.srl .\nginx\sslkeys\ca.srl
	copy .\auth\sslkeys\ca.srl .\dbservice\sslkeys\ca.srl


1Create-root-CA:
	openssl genpkey -algorithm RSA -out ./rootca/ca.key -aes256 -pass pass:standard_password
	openssl req -x509 -new -nodes -key ./rootca/ca.key -sha256 -days 3650 -out ./rootca/ca.crt -config ./rootca/ssl.cnf

2GeneratePrivate-key:
	openssl genpkey -algorithm RSA -out ./auth/sslkeys/server.key
	openssl genpkey -algorithm RSA -out ./dbservice/sslkeys/server.key
	openssl genpkey -algorithm RSA -out ./nginx/sslkeys/server.key

3Create-Certificate-Signing-Request:
	openssl req -new -key ./auth/sslkeys/server.key -out ./auth/sslkeys/server.csr -config ./rootca/ssl.cnf
	openssl req -new -key ./dbservice/sslkeys/server.key -out ./dbservice/sslkeys/server.csr -config ./rootca/ssl.cnf
	openssl req -new -key ./nginx/sslkeys/server.key -out ./nginx/sslkeys/server.csr -config ./rootca/ssl.cnf

4Sign-Server-Certificates-Using-RootCA:
	openssl x509 -req -in ./auth/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./auth/sslkeys/server.pem -days 365 -sha256 -extfile ./rootca/ssl.cnf -extensions req_ext
	openssl x509 -req -in ./dbservice/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./dbservice/sslkeys/server.pem -days 365 -sha256 -extfile ./rootca/ssl.cnf -extensions req_ext
	openssl x509 -req -in ./nginx/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./nginx/sslkeys/server.pem -days 365 -sha256 -extfile ./rootca/ssl.cnf -extensions req_ext

Mak:
	openssl genpkey -algorithm RSA -out ./rootca/ca.key
	openssl req -x509 -new -nodes -key ./rootca/ca.key -sha256 -days 365 -out ./rootca/ca.crt -subj "/C=US/ST=State/L=Locality/O=YourOrganization/CN=YourRootCA"

LL:
	openssl genpkey -algorithm RSA -out ./dbservice/sslkeys/server.key
	openssl req -new -key ./dbservice/sslkeys/server.key -out ./dbservice/sslkeys/server.csr -subj "/C=US/ST=State/L=Locality/O=YourOrganization/CN=localhost"
	openssl x509 -req -in ./dbservice/sslkeys/server.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./dbservice/sslkeys/server.crt -days 365 -sha256

Createtest:
	openssl genpkey -algorithm RSA -out ./dbservice/sslkeys/client.key
	openssl req -new -key ./dbservice/sslkeys/client.key -out ./dbservice/sslkeys/client.csr -subj "/C=US/ST=State/L=Locality/O=YourOrganization/CN=Client"
	openssl x509 -req -in ./dbservice/sslkeys/client.csr -CA ./rootca/ca.crt -CAkey ./rootca/ca.key -CAcreateserial -out ./dbservice/sslkeys/client.crt -days 365 -sha256


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