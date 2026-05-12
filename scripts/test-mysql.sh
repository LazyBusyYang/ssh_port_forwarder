#!/bin/bash
set -e

echo "Running MySQL integration tests..."

# Check if running in CI (GitHub Actions sets CI=true)
if [ -n "$CI" ]; then
  # In CI, MySQL service is provided by GitHub Actions
  export TEST_MYSQL_DSN="root:testroot@tcp(127.0.0.1:3306)/spf_test"
  go test ./internal/... -v -tags=integration -run=MySQL
else
  # Local development - start MySQL container
  echo "Starting MySQL container..."

  # Clean up any existing container
  docker stop spf-test-mysql 2>/dev/null || true
  docker rm spf-test-mysql 2>/dev/null || true

  docker run -d --name spf-test-mysql \
    -e MYSQL_ROOT_PASSWORD=testroot \
    -e MYSQL_DATABASE=spf_test \
    -p 3307:3306 \
    mysql:8.0

  echo "Waiting for MySQL to be ready..."
  for i in {1..30}; do
    if docker exec spf-test-mysql mysqladmin ping -uroot -ptestroot --silent 2>/dev/null; then
      echo "MySQL is ready"
      break
    fi
    echo "Waiting for MySQL... ($i/30)"
    sleep 2
  done

  export TEST_MYSQL_DSN="root:testroot@tcp(localhost:3307)/spf_test"
  go test ./internal/... -v -tags=integration -run=MySQL

  # Cleanup
  docker stop spf-test-mysql
  docker rm spf-test-mysql
fi

echo "MySQL tests passed!"
