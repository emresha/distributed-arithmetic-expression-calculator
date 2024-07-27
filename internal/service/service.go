package service

import (
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

var JWTSecretToken = []byte("please_dont_steal:(")

type Task struct {
	Id                  int    `json:"id"`
	Status              string `json:"status"`
	Original_Expression string `json:"original_expression"`
	Expression          string `json:"expression"`
	Result              int    `json:"result"`
	Owner               string `json:"owner"`
}

type Calculation struct {
	Task_id    int    `json:"task_id"`
	RPN_string string `json:"RPN_string"`
	Status     string `json:"status"`
	Result     int    `json:"result"`
}

type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func DeleteCalculationFromSlice(calc Calculation, calcSlice *[]Calculation) {
	for i := 0; i < len(*calcSlice); i++ {
		if calc.Task_id == (*calcSlice)[i].Task_id && calc.RPN_string == (*calcSlice)[i].RPN_string {
			*calcSlice = append((*calcSlice)[:i], (*calcSlice)[i+1:]...)
		}
	}
}

func CheckAuthentication(token *http.Cookie) (string, error) {
	tokenStr := token.Value
	claims := jwt.MapClaims{}
	JWT, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return JWTSecretToken, nil
	})

	if err != nil {
		return "", err
	}

	if !JWT.Valid {
		return "", errors.New("invalid token")
	}

	if name, ok := claims["name"].(string); ok {
		return name, nil
	}

	return "", errors.New("could extract name from token")
}
