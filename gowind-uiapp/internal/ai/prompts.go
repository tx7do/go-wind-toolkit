package ai

import (
	"sort"
	"strings"
)

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

// GetDDLFixPrompt DDL 校验失败后的修复重试提示:附上被否决的产物与解析错误。
func GetDDLFixPrompt(requirements string, previousDDL string, parseError string) string {
	return "Here are the requirements for a software system:\n\n" + requirements +
		"\n\nYou previously generated the following DDL, but it failed to parse:\n\n" + previousDDL +
		"\n\nThe parser reported:\n" + parseError +
		"\n\nPlease regenerate the COMPLETE corrected DDL (CREATE TABLE statements) that satisfies " +
		"the requirements and parses cleanly. Output ONLY raw SQL statements, no explanations."
}

// GetDDLConvertFixPrompt DDL 语法通过但生成管线(建表解析/列类型映射)拒绝时的
// 修复重试提示:与解析错误的提示区分开,明确失败发生在消费侧。
func GetDDLConvertFixPrompt(requirements string, previousDDL string, convertError string) string {
	return "Here are the requirements for a software system:\n\n" + requirements +
		"\n\nYou previously generated the following DDL. It parses, but the code generation pipeline cannot consume it:\n\n" + previousDDL +
		"\n\nThe pipeline reported:\n" + convertError +
		"\n\nPlease regenerate the COMPLETE corrected DDL (CREATE TABLE statements) that satisfies the requirements " +
		"and is fully consumable by the pipeline. Output ONLY raw SQL statements, no explanations."
}


// GetPartitionFixPrompt 划分结果无效后的修复重试提示:附上被否决的原始响应与问题。
func GetPartitionFixPrompt(ddl string, previousResponse string, problem string) string {
	return "Here is the database DDL:\n\n" + ddl +
		"\n\nYou previously responded with:\n\n" + previousResponse +
		"\n\nBut it has a problem: " + problem +
		"\n\nPlease analyze these database tables and partition them into microservices again. " +
		"Every table name must be taken verbatim from the DDL above. " +
		"Respond with ONLY a valid JSON array of {serviceName, tables, description}."
}

// 结构化审查系统提示:与自由散文版 reviewSystemPrompt 不同,要求只输出
// JSON 契约,供逐批审查的机读结果(渲染报告/跨版本 diff 都吃这个结构)。
const structuredReviewSystemPrompt = "You are an expert Go and microservice code reviewer. " +
	"Review the given batch of code files and report issues as structured JSON.\n\n" +
	"Focus on: correctness bugs, API design consistency, error handling, performance, " +
	"security, Go idioms, Kratos framework practices, proto/gRPC design.\n\n" +
	"You MUST respond with ONLY a valid JSON object, no markdown fences, no other text:\n" +
	`{"summary": "<one paragraph for this batch>", "issues": [{"severity": "critical|major|minor|info", "file": "<path as given>", "line": <int, 0 if unknown>, "title": "<short>", "detail": "<why and how to fix>"}]}` + "\n\n" +
	"Rules:\n" +
	"1. severity: critical = bugs/data loss/security; major = design or robustness flaws; minor = style/small improvements; info = observations.\n" +
	"2. file must be copied verbatim from the batch headers (=== path ===).\n" +
	"3. Report only real findings; an empty issues array is valid.\n" +
	"4. Write summary/detail in the same language as the code comments (Chinese if Chinese)."

// GetStructuredReviewPrompt 获取单批结构化审查的用户提示。
func GetStructuredReviewPrompt(fileContents map[string]string) string {
	// 复用散文版的文件拼接(=== path === 格式,GetReviewPrompt 内部按路径排序)
	return GetReviewPrompt(fileContents) +
		"\nRespond with ONLY the JSON object described in the system prompt."
}

// GetStructuredReviewFixPrompt 结构化审查解析失败后的修复重试提示。
func GetStructuredReviewFixPrompt(fileContents map[string]string, previousResponse string, problem string) string {
	return GetReviewPrompt(fileContents) +
		"\n\nYou previously responded with:\n\n" + previousResponse +
		"\n\nBut it has a problem: " + problem +
		"\n\nRespond again with ONLY the valid JSON object {\"summary\": ..., \"issues\": [...]}."
}

// GetReviewSynthesisPrompt 获取多批次汇总总评的用户提示。
func GetReviewSynthesisPrompt(batchSummaries []string) string {
	return "You reviewed a project in batches. Below are the per-batch outcomes " +
		"(summary and issue counts with the most important issues listed).\n\n" +
		strings.Join(batchSummaries, "\n\n") +
		"\n\nWrite a concise overall assessment in Chinese as Markdown: " +
		"overall health, the 2-4 most important issues to fix first (with file references), " +
		"and a short recommended action list. Do not repeat every issue."
}

// GetReviewPrompt 获取代码审查的完整用户提示
func GetReviewPrompt(fileContents map[string]string) string {
	// map 遍历序随机,先按路径排序保证提示词稳定可复现。
	paths := make([]string, 0, len(fileContents))
	for path := range fileContents {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	prompt := "Please review the following generated code files:\n\n"
	for _, path := range paths {
		prompt += "=== " + path + " ===\n"
		prompt += fileContents[path] + "\n\n"
	}
	prompt += "\nPlease provide your review and suggestions for improvement."
	return prompt
}
