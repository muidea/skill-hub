package template

import (
	"fmt"
	"regexp"
	"strings"
)

// VariablePattern 匹配模板变量的正则表达式
var VariablePattern = regexp.MustCompile(`\{\{\.(\w+)\}\}`)

// ExtractVariables 从模板内容中提取变量名
func ExtractVariables(template string) []string {
	matches := VariablePattern.FindAllStringSubmatch(template, -1)
	var variables []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			if !seen[varName] {
				variables = append(variables, varName)
				seen[varName] = true
			}
		}
	}

	return variables
}

// Render 渲染模板内容
func Render(template string, variables map[string]string) string {
	result := template
	for key, value := range variables {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// ReverseRender 尝试从渲染后的内容反向推导出模板
// 这是一个启发式算法，尝试将具体值替换回变量占位符
func ReverseRender(originalTemplate, renderedContent string, originalVariables map[string]string) (string, map[string]string) {
	// 提取原始模板中的变量
	templateVars := ExtractVariables(originalTemplate)

	// 如果模板没有变量，直接返回渲染后的内容
	if len(templateVars) == 0 {
		return renderedContent, originalVariables
	}

	// 创建一个映射，记录变量名到可能的值
	varValueCandidates := make(map[string][]string)

	// 对于每个变量，尝试在渲染后的内容中寻找匹配
	for _, varName := range templateVars {
		// 获取原始变量值
		originalValue, hasOriginal := originalVariables[varName]

		// 在渲染后的内容中搜索这个值
		if hasOriginal && originalValue != "" {
			// 检查原始值是否出现在渲染后的内容中
			if strings.Contains(renderedContent, originalValue) {
				varValueCandidates[varName] = []string{originalValue}
			}
		}
	}

	// 尝试构建新的模板
	newTemplate := originalTemplate
	updatedVariables := make(map[string]string)

	// 对于每个有候选值的变量，尝试替换回占位符
	for varName, candidates := range varValueCandidates {
		if len(candidates) > 0 {
			value := candidates[0]
			placeholder := "{{." + varName + "}}"

			// 检查这个值在渲染后的内容中是否唯一
			// 如果唯一，我们可以安全地替换回占位符
			occurrences := strings.Count(renderedContent, value)
			if occurrences == 1 {
				// 替换回占位符
				newTemplate = strings.Replace(newTemplate, placeholder, value, 1)
				// 保持变量值不变
				updatedVariables[varName] = value
			} else {
				// 多次出现，可能不是变量值，保持原样
				updatedVariables[varName] = value
			}
		} else {
			// 没有找到值，使用原始值
			if val, exists := originalVariables[varName]; exists {
				updatedVariables[varName] = val
			}
		}
	}

	// 对于没有找到值的变量，使用原始值
	for _, varName := range templateVars {
		if _, exists := updatedVariables[varName]; !exists {
			if val, exists := originalVariables[varName]; exists {
				updatedVariables[varName] = val
			}
		}
	}

	return newTemplate, updatedVariables
}

// DiffTemplates 比较两个模板的差异，返回差异行
func DiffTemplates(template1, template2 string) []string {
	lines1 := strings.Split(template1, "\n")
	lines2 := strings.Split(template2, "\n")

	var diffs []string
	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}

	for i := 0; i < maxLines; i++ {
		var line1, line2 string
		if i < len(lines1) {
			line1 = lines1[i]
		}
		if i < len(lines2) {
			line2 = lines2[i]
		}

		if line1 != line2 {
			diffs = append(diffs, fmt.Sprintf("Line %d: %q -> %q", i+1, line1, line2))
		}
	}

	return diffs
}

// SmartExtract 智能提取：从修改后的内容中提取变量值并更新模板
func SmartExtract(originalTemplate, modifiedContent string, currentVariables map[string]string) (string, map[string]string, error) {
	// 首先，用当前变量渲染原始模板
	renderedOriginal := Render(originalTemplate, currentVariables)

	// 如果修改后的内容与渲染后的原始内容相同，没有变化
	if strings.TrimSpace(modifiedContent) == strings.TrimSpace(renderedOriginal) {
		return originalTemplate, currentVariables, nil
	}

	// 提取原始模板中的变量
	templateVars := ExtractVariables(originalTemplate)

	// 如果没有变量，直接返回修改后的内容作为新模板
	if len(templateVars) == 0 {
		return modifiedContent, currentVariables, nil
	}

	// 简单启发式算法：尝试识别变量值的变化
	updatedVariables := make(map[string]string)

	// 复制当前变量
	for k, v := range currentVariables {
		updatedVariables[k] = v
	}

	// 对于每个变量，尝试在修改后的内容中寻找
	for _, varName := range templateVars {
		currentValue := currentVariables[varName]
		if currentValue == "" {
			continue
		}

		// 检查当前值是否出现在原始渲染内容中
		if strings.Contains(renderedOriginal, currentValue) {
			// 检查当前值是否出现在修改后的内容中
			if strings.Contains(modifiedContent, currentValue) {
				// 值没有变化，保持原样
				continue
			} else {
				// 值被修改或移除了
				// 尝试在修改后的内容中寻找新的值
				// 这是一个简化的实现，实际可能需要更复杂的算法
				fmt.Printf("变量 %s 的值可能被修改了\n", varName)
			}
		}
	}

	// 对于简单的文本替换，我们可以尝试一个更直接的方法
	// 如果修改只是变量值的变化，而不是模板结构的变化
	newTemplate := originalTemplate

	// 检查是否只是变量值的变化
	// 通过将修改后的内容与原始模板进行比较
	// 如果修改后的内容可以通过替换变量值得到，那么模板结构没有变化
	for _, varName := range templateVars {
		placeholder := "{{." + varName + "}}"
		// 如果占位符仍然在修改后的内容中，说明模板结构没有变化
		if strings.Contains(modifiedContent, placeholder) {
			// 模板结构没有变化，只是变量值可能变了
			// 保持模板不变
		} else {
			// 占位符被替换成了具体值，需要尝试恢复
			// 这是一个复杂的问题，暂时保持模板不变
			fmt.Printf("警告: 变量 %s 的占位符可能被替换了\n", varName)
		}
	}

	// 返回原始模板和当前变量（暂时不更新变量值）
	// 在实际使用中，用户应该手动更新变量值
	return newTemplate, updatedVariables, nil
}

// lineDiff 表示一行的差异
type lineDiff struct {
	lineNum  int
	original string
	modified string
}

// findLineDifferences 找出两段文本的行级差异
func findLineDifferences(original, modified []string) []lineDiff {
	var diffs []lineDiff
	maxLines := len(original)
	if len(modified) > maxLines {
		maxLines = len(modified)
	}

	for i := 0; i < maxLines; i++ {
		var origLine, modLine string
		if i < len(original) {
			origLine = original[i]
		}
		if i < len(modified) {
			modLine = modified[i]
		}

		if origLine != modLine {
			diffs = append(diffs, lineDiff{
				lineNum:  i + 1,
				original: origLine,
				modified: modLine,
			})
		}
	}

	return diffs
}

// isLikelyVariableValue 判断一行是否可能是变量值
func isLikelyVariableValue(line, currentValue string) bool {
	// 如果行是空的，可能不是变量值
	if strings.TrimSpace(line) == "" {
		return false
	}

	// 如果行包含占位符语法，可能不是具体的值
	if strings.Contains(line, "{{.") && strings.Contains(line, "}}") {
		return false
	}

	// 如果行与当前值相同，可能是变量值
	if line == currentValue {
		return true
	}

	// 启发式：如果行看起来像是一个具体的值（不是代码结构）
	// 这里可以添加更多启发式规则
	return true
}

// preserveVariables 在修改后的内容中尝试保留变量占位符
func preserveVariables(content string, variables []string, varValues map[string]string) string {
	result := content

	// 对于每个变量，如果它的值出现在内容中，尝试替换回占位符
	for _, varName := range variables {
		value, exists := varValues[varName]
		if exists && value != "" {
			placeholder := "{{." + varName + "}}"
			// 只在值出现一次时替换（避免误替换）
			if strings.Count(result, value) == 1 {
				result = strings.Replace(result, value, placeholder, 1)
			}
		}
	}

	return result
}
