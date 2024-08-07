package calculate

/*
This package is basically all the calculations logic
And I am not going to comment any of it since
It is overly complicated.
*/

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

var mu sync.Mutex


func IsFloat(s string) bool {
    _, err := strconv.ParseFloat(s, 64)
    return err == nil
}

func isOperator(char rune) bool {
    switch char {
    case '+', '-', '*', '/':
        return true
    }
    return false
}

// Validates an infix expression for correct syntax.

func tokenize(expr string) ([]string, error) {
	var tokens []string
	var buffer strings.Builder

	for _, ch := range expr {
		switch {
		case unicode.IsSpace(ch):
			continue
		case unicode.IsDigit(ch) || ch == '.':
			buffer.WriteRune(ch)
		case isOperator(ch) || ch == '(' || ch == ')':
			if buffer.Len() > 0 {
				tokens = append(tokens, buffer.String())
				buffer.Reset()
			}
			tokens = append(tokens, string(ch))
		default:
			return nil, fmt.Errorf("invalid character '%c'", ch)
		}
	}

	if buffer.Len() > 0 {
		tokens = append(tokens, buffer.String())
	}

	return tokens, nil
}

func ValidateInfixExpression(expr string) error {
	if strings.Contains(expr, ".") {
		return fmt.Errorf("invalid character '.'")
	}

	if len(expr) == 0 {
		return fmt.Errorf("got empty string")
	}

	tokens, err := tokenize(expr)
	if err != nil {
		return err
	}

	var balance int
	var lastToken string

	for i, token := range tokens {
		switch {
		case isOperator(rune(token[0])):
			if i == 0 || (i > 0 && isOperator(rune(lastToken[0])) && token != "-") || lastToken == "(" {
				return fmt.Errorf("operator %s at position %d is misplaced", token, i)
			}
		case token == "(":
			if i > 0 && (unicode.IsDigit(rune(lastToken[0])) || lastToken == ")" || isOperator(rune(lastToken[0]))) {
				return fmt.Errorf("missing operator before '(' at position %d", i)
			}
			balance++
		case token == ")":
			if balance == 0 {
				return fmt.Errorf("unmatched closing parenthesis at position %d", i)
			}
			if lastToken == "(" || isOperator(rune(lastToken[0])) {
				return fmt.Errorf("missing operand before ')' at position %d", i)
			}
			balance--
		default:
			// Token should be a number.
			if token == "-" && (i == 0 || lastToken == "(" || isOperator(rune(lastToken[0]))) {
				continue // Allow negative sign in these positions.
			}
			if _, err := strconv.ParseFloat(token, 64); err != nil {
				return fmt.Errorf("invalid token %s at position %d", token, i)
			}
			if i > 0 && (unicode.IsDigit(rune(lastToken[0])) || lastToken == ")") {
				return fmt.Errorf("missing operator before %s at position %d", token, i)
			}
		}
		lastToken = token
	}

	if balance != 0 {
		return fmt.Errorf("unmatched opening parenthesis")
	}

	if isOperator(rune(lastToken[0])) {
		return fmt.Errorf("expression ends with an operator")
	}

	return nil
}

func precedence(op rune) int {
    switch op {
    case '+', '-':
        return 1
    case '*', '/':
        return 2
    }
    return 0
}

func isLeftAssociative(op rune) bool {
    switch op {
    case '+', '-', '*', '/':
        return true
    }
    return false
}

func InfixToRPN(expression string) (string, error) {
	var output []string
	var operators []rune
	var buffer strings.Builder
	previousToken := ' '

	for _, token := range expression {
		switch {
		case unicode.IsSpace(token):
			continue // Ignore whitespace.
		case unicode.IsDigit(token) || (token == '-' && (previousToken == ' ' || previousToken == '(' || isOperator(previousToken))):
			// Handle negative numbers or digits.
			buffer.WriteRune(token)
		case unicode.IsLetter(token):
			buffer.WriteRune(token) // Accumulate variables.
		case token == '(':
			if buffer.Len() > 0 {
				output = append(output, buffer.String())
				buffer.Reset()
			}
			operators = append(operators, token)
		case token == ')':
			if buffer.Len() > 0 {
				output = append(output, buffer.String())
				buffer.Reset()
			}
			for len(operators) > 0 && operators[len(operators)-1] != '(' {
				output = append(output, string(operators[len(operators)-1]))
				operators = operators[:len(operators)-1]
			}
			if len(operators) == 0 {
				return "", fmt.Errorf("mismatched parentheses")
			}
			operators = operators[:len(operators)-1] // Pop the '('.
		case isOperator(token):
			if buffer.Len() > 0 {
				output = append(output, buffer.String())
				buffer.Reset()
			}
			for len(operators) > 0 && isOperator(operators[len(operators)-1]) &&
				((isLeftAssociative(token) && precedence(token) <= precedence(operators[len(operators)-1])) ||
					(!isLeftAssociative(token) && precedence(token) < precedence(operators[len(operators)-1]))) {
				output = append(output, string(operators[len(operators)-1]))
				operators = operators[:len(operators)-1]
			}
			operators = append(operators, token)
		}
		previousToken = token
	}

	if buffer.Len() > 0 {
		output = append(output, buffer.String())
	}

	for len(operators) > 0 {
		if operators[len(operators)-1] == '(' {
			return "", fmt.Errorf("mismatched parentheses")
		}
		output = append(output, string(operators[len(operators)-1]))
		operators = operators[:len(operators)-1]
	}

	return strings.Join(output, " "), nil
}


// This is my function from LeetCode :)
func EvalRPN(tokens []string) (int, error) {
	stack := []int{}
	for _, val := range tokens {
		n, err := strconv.Atoi(val)
		if err != nil {
			k := len(stack)
			switch val {
			case "+":
				new := stack[k-2] + stack[k-1]
				stack = stack[:k-2]
				stack = append(stack, new)
			case "-":
				new := stack[k-2] - stack[k-1]
				stack = stack[:k-2]
				stack = append(stack, new)

			case "*":
				new := stack[k-2] * stack[k-1]
				stack = stack[:k-2]
				stack = append(stack, new)
			case "/":

                if stack[k - 1] == 0 {
                    return 0, errors.New("DIVISION BY ZERO")
                }

				new := stack[k-2] / stack[k-1]
				stack = stack[:k-2]
				stack = append(stack, new)
			}
		} else {
			stack = append(stack, int(n))
		}

	}

	return int(stack[0]), nil
}


// this func evaluates the given RPN expression received from the expression channel
// and sends the result to the result slice.
func RPNtoSeparateCalculations(expression string, taskId int, db *sql.DB) {
	mu.Lock()
	defer mu.Unlock()
	tokens := strings.Split(expression, " ")

	for i := 2; i < len(tokens); i++ {
		if isOperator(rune(tokens[i][0])) && len(tokens[i]) == 1 {
			operand1 := tokens[i-2]
			operand2 := tokens[i-1]

			// Check if both operands are valid numbers.
			if _, err1 := strconv.ParseFloat(operand1, 64); err1 == nil {
				if _, err2 := strconv.ParseFloat(operand2, 64); err2 == nil {
					rpnString := operand1 + " " + operand2 + " " + tokens[i]
					status := "Waiting"
					result := 0

					// Check if the calculation already exists in the database.
					var count int
					err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE task_id = ? AND RPN_string = ?", taskId, rpnString).Scan(&count)
					if err != nil {
						log.Fatalf("Failed to check for existing calculation: %v\n", err)
					}
					if count > 0 {
						log.Printf("Calculation already exists: %s\n", rpnString)
						continue
					}

					// Insert the new calculation into the database.
					stmt, err := db.Prepare("INSERT INTO tasks(task_id, RPN_string, status, result) VALUES(?, ?, ?, ?)")
					if err != nil {
						log.Fatalf("Failed to prepare statement: %v\n", err)
					}
					defer stmt.Close()

					_, err = stmt.Exec(taskId, rpnString, status, result)
					if err != nil {
						log.Fatalf("Failed to execute statement: %v\n", err)
					}

					log.Printf("Inserted new calculation: %s\n", rpnString)
				} else {
					log.Printf("Invalid second operand: %s\n", operand2)
				}
			} else {
				log.Printf("Invalid first operand: %s\n", operand1)
			}
		} else {
			log.Printf("Token is not an operator or invalid operator length: %s\n", tokens[i])
		}
	}
}

func RPNtoInfix(expression string) (string, error) {
	tokens := strings.Split(expression, " ")
	stack := []string{}

	for _, token := range tokens {
		if len(token) == 0 {
			continue
		}
		if isOperator(rune(token[0])) && len(token) == 1 {
			// pop the last two operands from the stack
			if len(stack) < 2 {
				return "", fmt.Errorf("invalid RPN expression: not enough operands for operator %s", token)
			}
			operand2 := stack[len(stack)-1]
			operand1 := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			// create a new infix expression and push it back onto the stack
			newExpr := "(" + operand1 + " " + token + " " + operand2 + ")"
			stack = append(stack, newExpr)
		} else if isNumeric(token) || len(token) > 1 {
			// push operands onto the stack
			stack = append(stack, token)
		} else {
			return "", fmt.Errorf("invalid token in RPN expression: %s", token)
		}
	}

	// Handle single operand case
	if len(stack) == 1 {
		return stack[0], nil
	}

	// At the end, the stack should contain exactly one element, the final infix expression
	if len(stack) != 1 {
		return "", fmt.Errorf("invalid RPN expression: stack has %d elements after processing", len(stack))
	}

	return stack[0], nil
}

func isNumeric(token string) bool {
	for _, char := range token {
		if !unicode.IsDigit(char) && char != '.' {
			return false
		}
	}
	return true
}
