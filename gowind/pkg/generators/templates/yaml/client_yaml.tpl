client:
  grpc:
    timeout: 10s
    middleware:
      enable_logging: true
      enable_recovery: true
      enable_tracing: true
      enable_validate: true
      enable_circuit_breaker: true
      enable_metadata: true
      auth:
        method: "HS256"
        key: "<some_api_key>"