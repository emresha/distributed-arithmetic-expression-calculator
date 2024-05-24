package calculate

import (
	"distributed-calculator/internal/service"
	"fmt"
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
func ValidateInfixExpression(expr string) error {
    var balance int
    var lastChar rune

    for i, char := range expr {
        switch {
        case unicode.IsDigit(char):
            if i > 0 && (unicode.IsDigit(lastChar) || lastChar == ')') {
                return fmt.Errorf("missing operator before %c at position %d", char, i)
            }
        case char == '(':
            if i > 0 && (unicode.IsDigit(lastChar) || lastChar == ')') {
                return fmt.Errorf("missing operator before '(' at position %d", i)
            }
            balance++
        case char == ')':
            if balance == 0 {
                return fmt.Errorf("unmatched closing parenthesis at position %d", i)
            }
            if i > 0 && (lastChar == '(' || isOperator(lastChar)) {
                return fmt.Errorf("missing operand before ')' at position %d", i)
            }
            balance--
        case isOperator(char):
            if i == 0 || isOperator(lastChar) || lastChar == '(' {
                return fmt.Errorf("operator %c at position %d is misplaced", char, i)
            }
        case unicode.IsSpace(char):
            // allow spaces, do nothing
        default:
            return fmt.Errorf("invalid character %c at position %d", char, i)
        }
        lastChar = char
    }

    if balance != 0 {
        return fmt.Errorf("unmatched opening parenthesis")
    }

    if isOperator(lastChar) {
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

    for _, token := range expression {
        switch {
        case unicode.IsDigit(token):
            output = append(output, string(token))
        case token == '(':
            operators = append(operators, token)
        case token == ')':
            for len(operators) > 0 && operators[len(operators)-1] != '(' {
                output = append(output, string(operators[len(operators)-1]))
                operators = operators[:len(operators)-1]
            }
            if len(operators) == 0 {
                return "", fmt.Errorf("mismatched parentheses")
            }
            operators = operators[:len(operators)-1]
        case isOperator(token):
            for len(operators) > 0 && isOperator(operators[len(operators)-1]) &&
                ((isLeftAssociative(token) && precedence(token) <= precedence(operators[len(operators)-1])) ||
                    (!isLeftAssociative(token) && precedence(token) < precedence(operators[len(operators)-1]))) {
                output = append(output, string(operators[len(operators)-1]))
                operators = operators[:len(operators)-1]
            }
            operators = append(operators, token)
        }
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
func EvalRPN(tokens []string) int {
	stack := []int32{}
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
				new := stack[k-2] / stack[k-1]
				stack = stack[:k-2]
				stack = append(stack, new)
			}
		} else {
			stack = append(stack, int32(n))
		}

	}

	return int(stack[0])
}


// this func evaluates the given RPN expression received from the expression channel
// and sends the result to the result channel.
func RPNtoSeparateCalculations(expression string, taskId int, resultCh chan<- service.Calculation) {
	tokens := strings.Split(expression, " ")
	stack := []string{}

	for _, token := range tokens {
		if isOperator(rune(token[0])) {
			// pop the last two operands from the stack
			if len(stack) < 2 {
				continue 
                // not enough operands for this operator, skip
			}
			operand2 := stack[len(stack)-1]
			operand1 := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			// send the parallelizable part of the task to the channel
            newCalc := service.Calculation{
                Task_id: taskId,
                RPN_string: operand1 + " " + operand2 + " " + token,
                Status: "In Process",
                Result: 0,
            }
			resultCh <- newCalc
		} else {
			// push operands onto the stack
			stack = append(stack, token)
		}
	}

	close(resultCh)
}