// Generate a unique ID for each expression
function generateId() {
    return Math.floor(Math.random() * 1000000);
}

// Function to fetch and display tasks
async function getTasks() {
    const tasksContainer = document.getElementById('tasks-container');

    try {
        const response = await fetch('/api/v1/expressions');
        const tasks = await response.json();

        tasks.sort((a, b) => a.id - b.id);

        const existingTaskIds = new Set(Array.from(tasksContainer.children).map(child => parseInt(child.getAttribute('data-id'))));

        tasks.forEach(task => {
            let taskElement = document.querySelector(`[data-id="${task.id}"]`);

            if (taskElement) {
                // Update existing task if necessary
                const statusElement = taskElement.querySelector('.status');
                const expressionElement = taskElement.querySelector('.expression');
                const resultElement = taskElement.querySelector('.result');

                if (statusElement.innerText !== `Состояние: ${task.status}`) {
                    statusElement.innerText = `Состояние: ${task.status}`;
                    statusElement.className = `status ${task.status === 'Finished' ? 'finished' : ''}`;
                }
                if (expressionElement.innerText !== `Выражение: ${task.original_expression}`) {
                    expressionElement.innerText = `Выражение: ${task.original_expression}`;
                }
                if (resultElement.innerText !== `Результат: ${task.result}`) {
                    resultElement.innerText = `Результат: ${task.result}`;
                }
            } else {
                // Create new task element
                taskElement = document.createElement('div');
                taskElement.className = 'task';
                taskElement.setAttribute('data-id', task.id);
                taskElement.innerHTML = `
                    <p>ID Выражения: ${task.id}</p>
                    <p class="status ${task.status === 'Finished' ? 'finished' : ''}">Состояние: ${task.status}</p>
                    <p class="expression">Выражение: ${task.original_expression}</p>
                    <p class="result">Результат: ${task.result}</p>
                `;
                // Insert the new task at the beginning
                tasksContainer.insertBefore(taskElement, tasksContainer.firstChild);
            }

            existingTaskIds.delete(task.id);
        });

        // Remove outdated tasks
        existingTaskIds.forEach(id => {
            const outdatedTask = document.querySelector(`[data-id="${id}"]`);
            if (outdatedTask) {
                outdatedTask.remove();
            }
        });

    } catch (error) {
        console.error('Error fetching tasks:', error);
    }
}

// Handle form submission
document.querySelector('form').addEventListener('submit', async (event) => {
    event.preventDefault();
    const tasksContainer = document.getElementById('tasks-container');
    const expressionInput = document.getElementById('expression');
    const expression = expressionInput.value;
    const id = generateId();

    const data = {
        id: id,
        expression: expression
    };

    try {
        const response = await fetch('/api/v1/calculate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            // If the response status is not in the 200 range, add an error message
            const errorMessage = document.createElement('div');
            errorMessage.className = 'error';
            errorMessage.innerText = `Ошибка: Не удалось добавить выражение (статус: ${response.status}).`;
            tasksContainer.insertBefore(errorMessage, tasksContainer.firstChild);
        } else {
            expressionInput.value = '';
            getTasks();
        }
    } catch (error) {
        console.error('Error submitting expression:', error);
        // Add error message in case of network or other errors
        const errorMessage = document.createElement('div');
        errorMessage.className = 'task error';
        errorMessage.innerText = 'Ошибка: Не удалось добавить выражение (сетевая ошибка).';
        tasksContainer.insertBefore(errorMessage, tasksContainer.firstChild);
    }
});

// Call getTasks every 5 seconds (or a more reasonable interval)
setInterval(getTasks, 500);

// Initial call to display tasks on page load
getTasks();
