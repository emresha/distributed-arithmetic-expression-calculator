# Распределённый вычислитель арифметических выражений
Первый проект Яндекс.Лицея по языку Go.<br>
Очень сильно приветствуется обоснованное открытие Issues, Pull Request'ов и т.п.:)
## Условие
Пользователь хочет считать арифметические выражения. Он вводит строку `2 + 2 * 2` и хочет получить в ответ `6`. Но наши операции сложения и умножения (также деления и вычитания) выполняются "очень-очень" долго. Поэтому вариант, при котором пользователь делает http-запрос и получает в качестве ответа результат, невозможна. Более того, вычисление каждой такой операции в нашей "альтернативной реальности" занимает "гигантские" вычислительные мощности. Соответственно, каждое действие мы должны уметь выполнять отдельно и масштабировать эту систему можем добавлением вычислительных мощностей в нашу систему в виде новых "машин". Поэтому пользователь может с какой-то периодичностью уточнять у сервера "не посчиталость ли выражение"? Если выражение наконец будет вычислено - то он получит результат.
