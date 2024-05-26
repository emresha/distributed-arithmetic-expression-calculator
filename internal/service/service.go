package service

type Task struct {
	Id         int    `json:"id"`
	Status     string `json:"status"`
	Expression string `json:"expression"`
	Result     int    `json:"result"`
}

type Calculation struct {
	Task_id    int    `json:"task_id"`
	RPN_string string `json:"RPN_string"`
	Status     string `json:"status"`
	Result     int    `json:"result"`
}
