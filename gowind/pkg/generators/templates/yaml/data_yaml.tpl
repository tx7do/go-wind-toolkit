data:
  database:
    driver: "postgres"
    source: "host=postgres port=5432 user=postgres password=<your_password> dbname=<your_database> sslmode=disable"
#    driver: "mysql"
#    source: "root:<you_password>@tcp(localhost:3306)/<your_database>?parseTime=true&charset=utf8mb4&loc=Asia%2FShanghai"
    migrate: true
    debug: false
    enable_trace: false
    enable_metrics: false
    max_idle_connections: 25
    max_open_connections: 25
    connection_max_lifetime: 300s

  redis:
    addr: "redis:6379"
    password: "<your_password>"
    dial_timeout: 10s
    read_timeout: 0.4s
    write_timeout: 0.6s