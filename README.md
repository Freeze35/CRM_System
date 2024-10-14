Проект создан с целью предоставления упрощённую версию CRM (Customer Relationship Management) (В разработке)

CRM поможет автоматизировать систему управления в организации: 

- Платформа позволяет управлять задачами и проектами, назначать ответственных, отслеживать прогресс и сроки выполнения.

- Присутствует система управления задачами с возможностью разбиения на подзадачи, установления сроков и добавления комментариев.

- Инструменты для общения внутри команды: чат, форумы и внутренние социальные сети для организации

Проект основан на микросервисной архитектуре.  
В стек на данный момент входит:   

- Dokcer
- PostgressSQL
- Redis(кэширование)
- GRPC


### Запуск Backend на основе Makefile
    
#### Команда для сборки проекта на основе docker -compose 

    make up 

#### Команда для скрытия пула docker -compose

    make down 
    
Рекомендуемая установка make утилиты:  

**WINDOWS:**
1. Откройте PowerShell от имени администратора:  
Нажмите Win + X и выберите Windows PowerShell (Администратор) или Command Prompt (Администратор).
Разрешите запуск от имени администратора, если появится запрос.
2. Проверьте разрешение на выполнение скриптов:
   Выполните команду, чтобы убедиться, что вы можете запускать скрипты:  
   `Get-ExecutionPolicy`  
Если результат не равен RemoteSigned, установите его следующей командой:
   `Set-ExecutionPolicy RemoteSigned`  
3. Установите Chocolatey:
   Выполните следующую команду для установки Chocolatey:  
   `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))`  
4. Проверьте установку:  
      После завершения установки в той же сессии PowerShell выполните команду:  
   `choco --version`

Для установки make с помощью Chocolatey выполните следующие шаги:

1. Откройте Windows PowerShell от имени администратора:  
Нажмите Win + X и выберите Windows PowerShell (Администратор).  
2. Установите make с помощью команды Chocolatey:  
   `choco install make`
3. Подтвердите установку, введя Y, если появится запрос.  
4. После завершения установки проверьте, что make успешно установлен, выполнив команду:  
   `make --version`
5. Теперь make должен быть установлен на вашей системе, и вы можете его использовать.

#### PGAdmin
Для проверки работоспособности отдельного микросервиса Postgres рекомендуется:  
    Установить PGAdmin  
    https://www.pgadmin.org/download/pgadmin-4-windows/





    

    