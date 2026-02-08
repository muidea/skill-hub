# Logic Skill

A skill that demonstrates logical operations and decision making.

## Description

This skill provides logical operations including boolean logic, condition evaluation, and decision making capabilities. It's designed to be a simple yet comprehensive example of a skill that can be used for testing various skill-hub operations.

## Capabilities

### 1. Boolean Operations
- AND, OR, NOT operations
- XOR, NAND, NOR operations
- Truth table generation

### 2. Condition Evaluation
- Evaluate complex boolean expressions
- Handle nested conditions
- Support for multiple data types

### 3. Decision Making
- Simple if-then-else logic
- Pattern matching
- Rule-based decision systems

## Usage Examples

```python
# Example 1: Basic boolean operations
result = logic_and(True, False)  # Returns False
result = logic_or(True, False)   # Returns True

# Example 2: Complex condition evaluation
condition = "(A AND B) OR (C AND NOT D)"
result = evaluate_condition(condition, {"A": True, "B": False, "C": True, "D": True})

# Example 3: Decision making
decision = make_decision(rules, context)
```

## Input/Output Format

### Input
```json
{
  "operation": "and|or|not|xor|evaluate|decide",
  "operands": [true, false],
  "context": {
    "variables": {}
  }
}
```

### Output
```json
{
  "result": true,
  "explanation": "Operation completed successfully",
  "details": {}
}
```

## Error Handling

The skill handles:
- Invalid operation types
- Missing operands
- Type mismatches
- Syntax errors in expressions

## Testing

Run tests with:
```bash
python -m pytest tests/test_logic_skill.py
```

## Dependencies

- Python 3.8+
- No external dependencies

## License

MIT