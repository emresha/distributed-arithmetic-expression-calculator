package service

type Task struct {
	Id                  int    `json:"id"`
	Status              string `json:"status"`
	Original_Expression string `json:"original_expression"`
	Expression          string `json:"expression"`
	Result              int    `json:"result"`
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
