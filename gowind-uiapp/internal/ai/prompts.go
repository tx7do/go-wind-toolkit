package ai

// DDL 生成系统提示
const ddlSystemPrompt = "You are an expert database architect. Your task is to generate MySQL-compatible DDL (CREATE TABLE) statements based on the user's requirements.\n\nRules:\n1. Generate complete CREATE TABLE statements with proper column types, constraints, and indexes.\n2. Use MySQL-compatible syntax.\n3. Include PRIMARY KEY, NOT NULL, DEFAULT, AUTO_INCREMENT constraints where appropriate.\n4. Add proper indexes for foreign keys and commonly queried columns.\n5. Add COMMENT on tables and columns to describe their purpose.\n6. Use utf8mb4 character set.\n7. Follow naming conventions: table names in snake_case, use singular form (e.g., user, not users).\n8. Include created_at and updated_at TIMESTAMP columns for all tables.\n9. Include proper FOREIGN KEY constraints with ON DELETE and ON UPDATE rules.\n10. Output ONLY the SQL DDL statements, no explanations or markdown code fences.\n11. Do not wrap the output in ```sql``` code blocks."

// 微服务划分系统提示
const partitionSystemPrompt = "You are an expert microservice architect. Your task is to analyze database tables and suggest how to partition them into microservices.\n\nRules:\n1. Group related tables into logical microservices based on business domain.\n2. Each microservice should have a cohesive set of related tables.\n3. Consider data ownership, transaction boundaries, and domain boundaries.\n4. Suggest clear, descriptive service names in lowercase English (e.g., \"user\", \"order\", \"product\").\n5. You MUST respond with ONLY a valid JSON array, no other text.\n6. Each element must have: serviceName (string), tables (string array), description (string).\n7. The JSON should be directly parseable, NOT wrapped in markdown code fences.\n\nExample response format:\n[\n  {\"serviceName\": \"user\", \"tables\": [\"user\", \"user_role\", \"role\"], \"description\": \"用户和权限管理服务\"},\n  {\"serviceName\": \"order\", \"tables\": [\"order\", \"order_item\"], \"description\": \"订单管理服务\"}\n]"

// 代码审查系统提示
const reviewSystemPrompt = "You are an expert Go and microservice code reviewer. You will review generated code and provide constructive feedback.\n\nFocus on:\n1. Code correctness and potential bugs\n2. API design consistency\n3. Error handling patterns\n4. Performance considerations\n5. Security best practices\n6. Go idioms and conventions\n7. Kratos framework best practices\n8. Proto/gRPC service design\n\nProvide your feedback in a structured format with:\n- Issues found (if any)\n- Suggestions for improvement\n- Best practices that should be followed\n\nRespond in the same language as the user's input."

// GetDDLPrompt 获取 DDL 生成的完整用户提示
func GetDDLPrompt(requirements string) string {
	return "Here are the requirements for a software system:\n\n" + requirements +
		"\n\nPlease generate the complete database DDL (CREATE TABLE statements) for this system. " +
		"Output ONLY raw SQL statements, no explanations."
}

// GetPartitionPrompt 获取微服务划分的完整用户提示
func GetPartitionPrompt(ddl string) string {
	return "Here is the database DDL:\n\n" + ddl +
		"\n\nPlease analyze these database tables and suggest how to partition them into microservices. " +
		"Respond with ONLY a JSON array."
}

// GetReviewPrompt 获取代码审查的完整用户提示
func GetReviewPrompt(fileContents map[string]string) string {
	prompt := "Please review the following generated code files:\n\n"
	for path, content := range fileContents {
		prompt += "=== " + path + " ===\n"
		prompt += content + "\n\n"
	}
	prompt += "\nPlease provide your review and suggestions for improvement."
	return prompt
}
