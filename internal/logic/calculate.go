package calculate

import (
	"distributed-calculator/internal/service"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"
)

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

// validates an infix expression for correct syntax
func tokenize(expr string) ([]string, error) {
	var tokens []string
	var number []rune

	for i, char := range expr {
		switch {
		case unicode.IsDigit(char) || char == '.':
			number = append(number, char)
		case isOperator(char) || char == '(' || char == ')':
			if len(number) > 0 {
				tokens = append(tokens, string(number))
				number = []rune{}
			}
			tokens = append(tokens, string(char))
		case unicode.IsSpace(char):
			if len(number) > 0 {
				tokens = append(tokens, string(number))
				number = []rune{}
			}
		default:
			return nil, fmt.Errorf("invalid character %c at position %d", char, i)
		}
	}

	if len(number) > 0 {
		tokens = append(tokens, string(number))
	}

	return tokens, nil
}

func ValidateInfixExpression(expr string) error {
	tokens, err := tokenize(expr)
	if err != nil {
		return err
	}

	var balance int
	var lastToken string

	for i, token := range tokens {
		switch {
		case isOperator(rune(token[0])):
			if i == 0 || isOperator(rune(lastToken[0])) || lastToken == "(" {
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
			// Token should be a number
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

	for _, token := range expression {
		switch {
		case unicode.IsSpace(token):
			continue // Ignore whitespace
		case unicode.IsDigit(token):
			buffer.WriteRune(token) // Accumulate digits
		case unicode.IsLetter(token):
			buffer.WriteRune(token) // Accumulate variables
		case token == '(':
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
			operators = operators[:len(operators)-1] // Pop the '('
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
// and sends the result to the result channel.
func RPNtoSeparateCalculations(expression string, taskId int, resultCh chan<- service.Calculation) {
	tokens := strings.Split(expression, " ")
	stack := []string{}

	for _, token := range tokens {
		if len(token) == 0 {
			continue
		}
		if isOperator(rune(token[0])) && len(token) == 1 {
			// Check if there are exactly two operands in the stack
			if len(stack) != 2 {
				log.Println("Skipping operator due to incorrect number of operands:", token)
				continue // not exactly two operands for this operator, skip
			}
			operand2 := stack[len(stack)-1]
			operand1 := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			// Create a new calculation task
			newCalc := service.Calculation{
				Task_id:    taskId,
				RPN_string: operand1 + " " + operand2 + " " + token,
				Status:     "In Process",
				Result:     0,
			}
			log.Printf("NEW CALC: %s\n", newCalc.RPN_string)
			resultCh <- newCalc

		} else if isNumeric(token) {
			// Push operands onto the stack
			stack = append(stack, token)
		} else {
			log.Println("Error: Invalid token", token)
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
