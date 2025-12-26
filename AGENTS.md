# AGENTS.md

## Project rules

- Use English to write code and comments.
- Add comments for each function and struct to help developers understand their purpose.
- Use the slog package for all logging purposes.
- Prefer using the `any` keyword instead of `interface{}` for empty interfaces.
- Avoid using `fmt.Sprint` or `fmt.Sprintf` for simple string concatenation in performance-critical code; use efficient alternatives like direct concatenation or the `strconv` package.