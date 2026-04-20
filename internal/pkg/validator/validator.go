package validator

import "fmt"

// ValidatePortRange 校验端口是否在允许范围内
func ValidatePortRange(port, min, max int) error {
	if port < min || port > max {
		return fmt.Errorf("port %d is out of allowed range [%d, %d]", port, min, max)
	}
	return nil
}

// ValidateRequired 校验必填字段
func ValidateRequired(field string, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// ValidateAuthMethod 校验认证方式
func ValidateAuthMethod(method string) error {
	if method != "password" && method != "private_key" {
		return fmt.Errorf("auth_method must be 'password' or 'private_key', got '%s'", method)
	}
	return nil
}

// ValidateStrategy 校验 LB 策略
func ValidateStrategy(strategy string) error {
	switch strategy {
	case "round_robin", "least_rules", "weighted":
		return nil
	default:
		return fmt.Errorf("strategy must be 'round_robin', 'least_rules', or 'weighted', got '%s'", strategy)
	}
}
